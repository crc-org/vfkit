package test

import (
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/crc-org/vfkit/pkg/config"

	vfkithelpers "github.com/crc-org/crc/v2/pkg/drivers/vfkit"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func retryIPFromMAC(macAddress string) (string, error) {
	var (
		err error
		ip  string
	)

	for i := 0; i < 100; i++ {
		ip, err = vfkithelpers.GetIPAddressByMACAddress(macAddress)
		if err == nil {
			break
		}

		time.Sleep(time.Second)
	}
	if err != nil {
		return "", err
	}
	log.Infof("found IP address %s for MAC %s", ip, macAddress)

	return ip, nil
}

func retrySSHDial(scheme string, address string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	for i := 0; i < 10; i++ {
		log.Debugf("ssh dial try #%d", i)
		sshClient, err := ssh.Dial(scheme, address, sshConfig)
		if err == nil {
			log.Infof("established SSH connection to %s over %s", address, scheme)
			return sshClient, nil
		}
		log.Debugf("ssh failed: %v", err)
		time.Sleep(time.Second)
	}
	return nil, fmt.Errorf("timeout waiting for SSH")
}

type vfkitRunner struct {
	*exec.Cmd
	gracefullyShutdown bool
}

func startVfkit(t *testing.T, vm *config.VirtualMachine) *vfkitRunner {
	const vfkitRelativePath = "../out/vfkit"

	binaryPath, err := exec.LookPath(vfkitRelativePath)
	require.NoError(t, err)

	log.Infof("starting %s", binaryPath)
	vfkitCmd, err := vm.Cmd(binaryPath)
	require.NoError(t, err)

	err = vfkitCmd.Start()
	require.NoError(t, err)

	return &vfkitRunner{
		vfkitCmd,
		false,
	}
}

func (cmd *vfkitRunner) Wait(t *testing.T) {
	err := cmd.Cmd.Wait()
	require.NoError(t, err)
	cmd.gracefullyShutdown = true
}

func (cmd *vfkitRunner) Close() {
	if cmd != nil && !cmd.gracefullyShutdown {
		log.Infof("killing left-over vfkit process")
		err := cmd.Cmd.Process.Kill()
		if err != nil {
			log.Warnf("failed to kill vfkit process: %v", err)
		}
	}
}

type testVM struct {
	provider OsProvider
	config   *config.VirtualMachine

	sshNetwork string
	macAddress string // for SSH over TCP
	port       int
	vsockPath  string // for SSH over vsock
	sshClient  *ssh.Client

	vfkitCmd *vfkitRunner
}

func NewTestVM(t *testing.T, provider OsProvider) *testVM { //nolint:revive
	vm := &testVM{
		provider: provider,
	}
	vmConfig, err := provider.ToVirtualMachine()
	require.NoError(t, err)
	vm.config = vmConfig

	return vm
}

func (vm *testVM) findSSHAccessMethod(t *testing.T, network string) *SSHAccessMethod {
	switch network {
	case "any":
		accessMethods := vm.provider.SSHAccessMethods()
		require.NotZero(t, len(accessMethods))
		return &accessMethods[0]
	default:
		for _, accessMethod := range vm.provider.SSHAccessMethods() {
			if accessMethod.network == network {
				return &accessMethod
			}
		}
	}

	t.FailNow()
	return nil
}

func (vm *testVM) AddSSH(t *testing.T, network string) {
	const vmMacAddress = "56:46:4b:49:54:01"
	var (
		dev config.VirtioDevice
		err error
	)
	method := vm.findSSHAccessMethod(t, network)
	switch network {
	case "tcp":
		log.Infof("adding virtio-net device (MAC: %s)", vmMacAddress)
		vm.sshNetwork = "tcp"
		vm.macAddress = vmMacAddress
		vm.port = method.port
		dev, err = config.VirtioNetNew(vm.macAddress)
		require.NoError(t, err)
	case "vsock":
		log.Infof("adding virtio-vsock device (port: %d)", method.port)
		vm.sshNetwork = "vsock"
		vm.vsockPath = filepath.Join(t.TempDir(), fmt.Sprintf("vsock-%d.sock", method.port))
		dev, err = config.VirtioVsockNew(uint(method.port), vm.vsockPath, false)
		require.NoError(t, err)
	default:
		t.FailNow()
	}

	vm.AddDevice(t, dev)
}

func (vm *testVM) AddDevice(t *testing.T, dev config.VirtioDevice) {
	err := vm.config.AddDevice(dev)
	require.NoError(t, err)
}

func (vm *testVM) Start(t *testing.T) {
	vm.vfkitCmd = startVfkit(t, vm.config)
}

func (vm *testVM) Stop(t *testing.T) {
	vm.SSHRun(t, "poweroff")
	vm.vfkitCmd.Wait(t)
}

func (vm *testVM) Close(_ *testing.T) {
	if vm.sshClient != nil {
		vm.sshClient.Close()
	}
	vm.vfkitCmd.Close()
}

func (vm *testVM) WaitForSSH(t *testing.T) {
	var (
		sshClient *ssh.Client
		err       error
	)
	switch vm.sshNetwork {
	case "tcp":
		ip, err := retryIPFromMAC(vm.macAddress)
		require.NoError(t, err)
		sshClient, err = retrySSHDial("tcp", net.JoinHostPort(ip, strconv.Itoa(vm.port)), vm.provider.SSHConfig())
		require.NoError(t, err)
	case "vsock":
		sshClient, err = retrySSHDial("unix", vm.vsockPath, vm.provider.SSHConfig())
		require.NoError(t, err)
	default:
		t.FailNow()
	}

	vm.sshClient = sshClient
}

func (vm *testVM) SSHRun(t *testing.T, command string) {
	sshSession, err := vm.sshClient.NewSession()
	require.NoError(t, err)
	defer sshSession.Close()
	_ = sshSession.Run(command)
}

func (vm *testVM) SSHCombinedOutput(t *testing.T, command string) ([]byte, error) {
	sshSession, err := vm.sshClient.NewSession()
	require.NoError(t, err)
	defer sshSession.Close()
	return sshSession.CombinedOutput(command)
}
