package test

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/crc-org/vfkit/pkg/config"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

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
		createDev: func(t *testing.T) (config.VirtioDevice, error) {
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
		createDev: func(t *testing.T) (config.VirtioDevice, error) {
			return config.VirtioRngNew()
		},
	},
	"virtio-fs": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x105a,
		createDev: func(t *testing.T) (config.VirtioDevice, error) {
			return config.VirtioFsNew("./", "vfkit-share-test")
		},
	},
	"virtio-gpu": {
		vendorID: 0x1af4, // Red Hat
		deviceID: 0x1050,
		createDev: func(t *testing.T) (config.VirtioDevice, error) {
			return config.VirtioGPUNew()
		},
	},
	"virtio-input/pointing-device": {
		vendorID: 0x106b, // Apple
		deviceID: 0x1a06,
		createDev: func(t *testing.T) (config.VirtioDevice, error) {
			return config.VirtioInputNew("pointing")
		},
	},
	"virtio-input/keyboard": {
		vendorID: 0x106b, // Apple
		deviceID: 0x1a06,
		createDev: func(t *testing.T) (config.VirtioDevice, error) {
			return config.VirtioInputNew("keyboard")
		},
	},
}

func TestPCIIds(t *testing.T) {
	puipuiProvider := NewPuipuiProvider()
	log.Info("fetching os image")
	tempDir := t.TempDir()
	err := puipuiProvider.Fetch(tempDir)
	require.NoError(t, err)

	for name, test := range pciidTests {
		t.Run(name, func(t *testing.T) {
			vm := NewTestVM(t, puipuiProvider)
			defer vm.Close(t)
			require.NotNil(t, vm)

			vm.AddSSH(t, "tcp")
			dev, err := test.createDev(t)
			require.NoError(t, err)
			vm.AddDevice(t, dev)

			vm.Start(t)
			vm.WaitForSSH(t)
			checkPCIDevice(t, vm, test.vendorID, test.deviceID)
			vm.Stop(t)
		})
	}
}

func checkPCIDevice(t *testing.T, vm *testVM, vendorID, deviceID int) {
	re := regexp.MustCompile(fmt.Sprintf("(?m)[[:blank:]]%04x:%04x\n", vendorID, deviceID))
	lspci, err := vm.SSHCombinedOutput(t, "lspci")
	log.Infof("lspci: %s", string(lspci))
	require.NoError(t, err)
	require.Regexp(t, re, string(lspci))
}
