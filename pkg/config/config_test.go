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
