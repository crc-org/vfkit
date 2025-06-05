package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddIgnitionFile_MultipleOptions(t *testing.T) {
	vm := &VirtualMachine{}
	err := vm.AddIgnitionFileFromCmdLine("file1,file2")
	assert.EqualError(t, err, "ignition only accept one option in command line argument")
}

func TestAddIgnitionFile_OneOption(t *testing.T) {
	vm := &VirtualMachine{}
	err := vm.AddIgnitionFileFromCmdLine("file1")
	require.NoError(t, err)
	assert.Len(t, vm.Devices, 1)
	assert.Equal(t, uint32(ignitionVsockPort), vm.Devices[0].(*VirtioVsock).Port)
	assert.Equal(t, "file1", vm.Ignition.ConfigPath)
}

func TestNetworkBlockDevice(t *testing.T) {
	vm := &VirtualMachine{}
	gpu, _ := VirtioGPUNew()
	vm.Devices = append(vm.Devices, gpu)
	nbd, _ := NetworkBlockDeviceNew("uri", 1000, SynchronizationFullMode)
	nbd.DeviceIdentifier = "nbd1"
	vm.Devices = append(vm.Devices, nbd)
	nbd2, _ := NetworkBlockDeviceNew("uri2", 1000, SynchronizationFullMode)
	nbd2.DeviceIdentifier = "nbd2"
	vm.Devices = append(vm.Devices, nbd2)

	nbdItem := vm.NetworkBlockDevice("nbd2")
	assert.Equal(t, "nbd2", nbdItem.DeviceIdentifier)
	assert.Equal(t, "uri2", nbdItem.URI)
}

func TestNetworkBlockDevice_NoDevice(t *testing.T) {
	vm := &VirtualMachine{}

	nbdItem := vm.NetworkBlockDevice("nbd2")
	require.Nil(t, nbdItem)
}

func TestVirtualMachine_ValidateBlockDevices(t *testing.T) {
	vm := &VirtualMachine{}

	tmpDir := t.TempDir()
	imagePath := filepath.Join(tmpDir, "disk.qcow2")
	size := "1G"

	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", imagePath, size)
	err := cmd.Run()

	require.NoError(t, err)
	defer os.Remove(imagePath)

	dev, err := VirtioBlkNew(imagePath)
	require.NoError(t, err)
	vm.Devices = append(vm.Devices, dev)

	err = dev.validate()
	require.Error(t, err)

	require.ErrorContains(t, err, "vfkit does not support qcow2 image format")
}
