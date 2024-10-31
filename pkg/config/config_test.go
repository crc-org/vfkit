package config

import (
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
