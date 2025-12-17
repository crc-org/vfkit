package test

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/crc-org/vfkit/pkg/config"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailedVfkitStart(t *testing.T) {
	puipuiProvider := NewPuipuiProvider()
	log.Info("fetching os image")
	err := puipuiProvider.Fetch(t.TempDir())
	require.NoError(t, err)

	vm := NewTestVM(t, puipuiProvider)
	defer vm.Close(t)
	require.NotNil(t, vm)

	vm.AddSSH(t, "vsock")

	dev, err := config.NVMExpressControllerNew("/a/b")
	require.NoError(t, err)
	vm.AddDevice(t, dev)

	vm.Start(t)

	log.Infof("waiting for SSH")
	_, err = retrySSHDial(vm.vfkitCmd.errCh, "unix", vm.vsockPath, vm.provider.SSHConfig())
	require.Error(t, err)
}

func testSSHAccess(t *testing.T, vm *testVM, network string) {
	log.Infof("testing SSH access over %s", network)
	vm.AddSSH(t, network)
	vm.Start(t)

	log.Infof("waiting for SSH")
	vm.WaitForSSH(t)

	log.Infof("shutting down VM")
	vm.Stop(t)
}

func TestSSHAccess(t *testing.T) {
	puipuiProvider := NewPuipuiProvider()
	log.Info("fetching os image")
	err := puipuiProvider.Fetch(t.TempDir())
	require.NoError(t, err)

	for _, accessMethod := range puipuiProvider.SSHAccessMethods() {
		t.Run(accessMethod.network, func(t *testing.T) {
			vm := NewTestVM(t, puipuiProvider)
			defer vm.Close(t)
			require.NotNil(t, vm)
			testSSHAccess(t, vm, accessMethod.network)
		})
	}
}

// guest listens over vsock, host connects to the guest
func TestVsockConnect(t *testing.T) {
	puipuiProvider := NewPuipuiProvider()
	log.Info("fetching os image")
	err := puipuiProvider.Fetch(t.TempDir())
	require.NoError(t, err)

	vm := NewTestVM(t, puipuiProvider)
	defer vm.Close(t)
	require.NotNil(t, vm)

	vm.AddSSH(t, "tcp")

	tempDir := t.TempDir()
	vsockConnectPath := filepath.Join(tempDir, "vsock-connect.sock")
	dev, err := config.VirtioVsockNew(1234, vsockConnectPath, false)
	require.NoError(t, err)
	vm.AddDevice(t, dev)

	vm.Start(t)
	vm.WaitForSSH(t)

	log.Infof("path to vsock socket: %s", vsockConnectPath)
	go func() {
		for i := 0; i < 5; i++ {
			conn, err := net.DialTimeout("unix", vsockConnectPath, time.Second)
			require.NoError(t, err)
			defer conn.Close()
			data, err := io.ReadAll(conn)
			require.NoError(t, err)
			if len(data) != 0 {
				log.Infof("read data from guest: %v", string(data))
				require.Equal(t, []byte("hello host"), data)
				break
			}
		}
	}()
	log.Infof("running socat")
	vm.SSHRun(t, "echo -n 'hello host' | socat - VSOCK-LISTEN:1234")

	log.Infof("stopping VM")
	vm.Stop(t)
}

// host listens over vsock, guest connects to the host
func TestVsockListen(t *testing.T) {
	puipuiProvider := NewPuipuiProvider()
	log.Info("fetching os image")
	err := puipuiProvider.Fetch(t.TempDir())
	require.NoError(t, err)

	vm := NewTestVM(t, puipuiProvider)
	defer vm.Close(t)
	require.NotNil(t, vm)

	vm.AddSSH(t, "tcp")

	tempDir := t.TempDir()
	vsockListenPath := filepath.Join(tempDir, "vsock-listen.sock")
	ln, err := net.Listen("unix", vsockListenPath)
	require.NoError(t, err)
	go func() {
		conn, err := ln.Accept()
		// call ln.Close() after a timeout to unblock Accept() and fail the test?
		require.NoError(t, err)
		data, err := io.ReadAll(conn)
		require.NoError(t, err)
		log.Infof("read %v", string(data))
		require.Equal(t, []byte("hello host"), data)
	}()
	log.Infof("path to vsock socket: %s", vsockListenPath)
	dev, err := config.VirtioVsockNew(1235, vsockListenPath, true)
	require.NoError(t, err)
	vm.AddDevice(t, dev)

	vm.Start(t)
	vm.WaitForSSH(t)

	vm.SSHRun(t, "echo -n 'hello host' | socat -T 2 STDIN VSOCK-CONNECT:2:1235")

	vm.Stop(t)
}

func TestFileSharing(t *testing.T) {
	puipuiProvider := NewPuipuiProvider()
	log.Info("fetching os image")
	tempDir := t.TempDir()
	err := puipuiProvider.Fetch(tempDir)
	require.NoError(t, err)

	vm := NewTestVM(t, puipuiProvider)
	defer vm.Close(t)
	require.NotNil(t, vm)

	vm.AddSSH(t, "tcp")

	sharedDir := t.TempDir()
	share, err := config.VirtioFsNew(sharedDir, "vfkit-test-share")
	require.NoError(t, err)
	vm.AddDevice(t, share)
	log.Infof("shared directory: %s", sharedDir)

	vm.Start(t)
	vm.WaitForSSH(t)

	vm.SSHRun(t, "mkdir /mnt")
	vm.SSHRun(t, "mount -t virtiofs vfkit-test-share /mnt")

	err = os.WriteFile(filepath.Join(sharedDir, "from-host.txt"), []byte("data from host"), 0600)
	require.NoError(t, err)
	data, err := vm.SSHCombinedOutput(t, "cat /mnt/from-host.txt")
	require.NoError(t, err)
	require.Equal(t, "data from host", string(data))

	vm.SSHRun(t, "echo -n 'data from guest' > /mnt/from-guest.txt")
	data, err = os.ReadFile(filepath.Join(sharedDir, "from-guest.txt"))
	require.NoError(t, err)
	require.Equal(t, "data from guest", string(data))

	vm.Stop(t)
}

type createDevFunc func(t *testing.T) (config.VirtioDevice, error)
type pciidTest struct {
	vendorID  int
	deviceID  int
	createDev createDevFunc
}

var pciidTests = map[string]pciidTest{
	"virtio-net": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x1041,
		createDev: func(_ *testing.T) (config.VirtioDevice, error) {
			return config.VirtioNetNew("")
		},
	},
	"virtio-serial": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x1043,
		createDev: func(t *testing.T) (config.VirtioDevice, error) {
			return config.VirtioSerialNew(filepath.Join(t.TempDir(), "serial.log"))
		},
	},
	"virtio-rng": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x1044,
		createDev: func(_ *testing.T) (config.VirtioDevice, error) {
			return config.VirtioRngNew()
		},
	},
	"virtio-fs": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x105a,
		createDev: func(_ *testing.T) (config.VirtioDevice, error) {
			return config.VirtioFsNew("./", "vfkit-share-test")
		},
	},
	"virtio-balloon": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x1045,
		createDev: func(_ *testing.T) (config.VirtioDevice, error) {
			return config.VirtioBalloonNew()
		},
	},
}

var pciidMacOS13Tests = map[string]pciidTest{
	"virtio-gpu": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x1050,
		createDev: func(_ *testing.T) (config.VirtioDevice, error) {
			return config.VirtioGPUNew()
		},
	},
	"virtio-input/trackpad": {
		vendorID: 0x106b, // Apple
		deviceID: 0x1a06,
		createDev: func(_ *testing.T) (config.VirtioDevice, error) {
			return config.VirtioInputNew("pointing")
		},
	},
	"virtio-input/keyboard": {
		vendorID: 0x106b, // Apple
		deviceID: 0x1a06,
		createDev: func(_ *testing.T) (config.VirtioDevice, error) {
			return config.VirtioInputNew("keyboard")
		},
	},
}

var pciidMacOS14Tests = map[string]pciidTest{
	"nvm-express": {
		vendorID: 0x106b, // Apple
		deviceID: 0x1a09,
		createDev: func(t *testing.T) (config.VirtioDevice, error) {
			diskimg := filepath.Join(t.TempDir(), "nvmexpress.img")
			f, err := os.Create(diskimg)
			require.NoError(t, err)
			require.NoError(t, f.Close())
			return config.NVMExpressControllerNew(diskimg)
		},
	},
}

var pciidVersionedTests = map[int]map[string]pciidTest{
	13: pciidMacOS13Tests,
	14: pciidMacOS14Tests,
}

func restInspect(t *testing.T, vm *testVM) *config.VirtualMachine {
	tr := &http.Transport{
		Dial: func(_, _ string) (conn net.Conn, err error) {
			return net.Dial("unix", vm.restSocketPath)
		},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get("http://vfkit/vm/inspect")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var unmarshalledVM config.VirtualMachine
	err = json.Unmarshal(body, &unmarshalledVM)
	require.NoError(t, err)
	return &unmarshalledVM
}

func testPCIId(t *testing.T, test pciidTest, provider OsProvider) {
	vm := NewTestVM(t, provider)
	defer vm.Close(t)
	require.NotNil(t, vm)

	vm.AddSSH(t, "tcp")
	dev, err := test.createDev(t)
	require.NoError(t, err)
	vm.AddDevice(t, dev)

	vm.Start(t)
	vm.WaitForSSH(t)
	checkPCIDevice(t, vm, test.vendorID, test.deviceID)

	unmarshalledVM := restInspect(t, vm)
	require.Equal(t, vm.config, unmarshalledVM)

	vm.Stop(t)
}

func TestPCIIds(t *testing.T) {
	puipuiProvider := NewPuipuiProvider()
	log.Info("fetching os image")
	tempDir := t.TempDir()
	err := puipuiProvider.Fetch(tempDir)
	require.NoError(t, err)

	for name, test := range pciidTests {
		t.Run(name, func(t *testing.T) {
			testPCIId(t, test, puipuiProvider)
		})
	}

	for macosVersion, tests := range pciidVersionedTests {
		if err := macOSAvailable(float64(macosVersion)); err == nil {
			for name, test := range tests {
				t.Run(name, func(t *testing.T) {
					testPCIId(t, test, puipuiProvider)
				})
			}
		} else {
			t.Logf("Skipping macOS %d tests", macosVersion)
		}
	}
}

func TestVirtioSerialPTY(t *testing.T) {
	puipuiProvider := NewPuipuiProvider()
	log.Info("fetching os image")
	tempDir := t.TempDir()
	err := puipuiProvider.Fetch(tempDir)
	require.NoError(t, err)

	vm := NewTestVM(t, puipuiProvider)
	defer vm.Close(t)
	require.NotNil(t, vm)

	vm.AddSSH(t, "tcp")
	dev, err := config.VirtioSerialNewPty()
	require.NoError(t, err)
	vm.AddDevice(t, dev)

	vm.Start(t)
	vm.WaitForSSH(t)
	runtimeVM := restInspect(t, vm)
	var foundVirtioSerial bool
	for _, dev := range runtimeVM.Devices {
		runtimeDev, ok := dev.(*config.VirtioSerial)
		if ok {
			assert.NotEmpty(t, runtimeDev.PtyName)
			foundVirtioSerial = true
			break
		}
	}
	require.True(t, foundVirtioSerial)

	vm.Stop(t)
}

func checkPCIDevice(t *testing.T, vm *testVM, vendorID, deviceID int) {
	re := regexp.MustCompile(fmt.Sprintf("(?m)[[:blank:]]%04x:%04x\n", vendorID, deviceID))
	lspci, err := vm.SSHCombinedOutput(t, "lspci")
	log.Infof("lspci: %s", string(lspci))
	require.NoError(t, err)
	require.Regexp(t, re, string(lspci))
}

func TestCloudInit(t *testing.T) {
	if err := macOSAvailable(13); err != nil {
		t.Log("Skipping TestCloudInit test")
		return
	}
	fedoraProvider := NewFedoraProvider()
	log.Info("fetching os image")
	tempDir := t.TempDir()
	err := fedoraProvider.Fetch(tempDir)
	require.NoError(t, err)

	// set efi bootloader
	fedoraProvider.efiVariableStorePath = "efi-variable-store"
	fedoraProvider.createVariableStore = true

	vm := NewTestVM(t, fedoraProvider)
	defer vm.Close(t)
	require.NotNil(t, vm)

	vm.AddSSH(t, "tcp")

	// add vm image
	dev1, err := config.VirtioBlkNew(fedoraProvider.diskImage)
	require.NoError(t, err)
	vm.AddDevice(t, dev1)
	log.Infof("shared disk: %s - fedora", dev1.DevName)

	/* 	add cloud init config by using a premade ISO image
	   	seed.img is an ISO image containing the user-data and meta-data file needed to configure the VM by cloud-init.
	   	meta-data is an empty file
	   	user-data has info about a new user that will be used to verify if the configuration has been applied. Its content is
		----
	   	#cloud-config
		users:
		- name: vfkituser
			sudo: ALL=(ALL) NOPASSWD:ALL
			shell: /bin/bash
			groups: users
			plain_text_passwd: vfkittest
			lock_passwd: false
		ssh_pwauth: true
		chpasswd: { expire: false }
	*/
	dev, err := config.VirtioBlkNew("assets/seed.img")
	require.NoError(t, err)
	vm.AddDevice(t, dev)
	log.Infof("shared disk: %s - cloud-init", dev.DevName)

	vm.Start(t)
	vm.WaitForSSH(t)

	data, err := vm.SSHCombinedOutput(t, "whoami")
	require.NoError(t, err)
	log.Infof("executed whoami - output: %s", string(data))
	require.Equal(t, "vfkituser\n", string(data))

	log.Info("stopping vm")
	vm.Stop(t)
}

// host listens over vsock, guest connects to the host
func TestIgnition(t *testing.T) {
	const ignTestData = "ignition config test\n"
	puipuiProvider := NewPuipuiProvider()
	log.Info("fetching os image")
	err := puipuiProvider.Fetch(t.TempDir())
	require.NoError(t, err)

	vm := NewTestVM(t, puipuiProvider)
	defer vm.Close(t)
	require.NotNil(t, vm)

	ignConfigPath := filepath.Join(t.TempDir(), "config.ign")
	err = os.WriteFile(ignConfigPath, []byte(ignTestData), 0600)
	require.NoError(t, err)
	vm.AddIgnition(t, ignConfigPath)

	vm.AddSSH(t, "tcp")

	vm.Start(t)
	vm.WaitForSSH(t)

	output, err := vm.SSHCombinedOutput(t, "socat -T 2 TCP-LISTEN:8080 VSOCK-CONNECT:2:1024 >/dev/null & curl -q http://localhost:8080")
	require.NoError(t, err)
	require.Contains(t, string(output), ignTestData)

	// time.Sleep(3600 * time.Second)
	vm.Stop(t)
}
