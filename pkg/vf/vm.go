package vf

import (
	"fmt"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
)

func ToVzVirtualMachineConfig(vm *config.VirtualMachine) (*vz.VirtualMachineConfiguration, error) {
	vzBootloader, err := ToVzBootloader(vm.Bootloader)
	if err != nil {
		return nil, err
	}

	vzVMConfig, err := vz.NewVirtualMachineConfiguration(vzBootloader, vm.Vcpus, vm.MemoryBytes)
	if err != nil {
		return nil, err
	}

	for _, dev := range vm.Devices {
		if err := AddToVirtualMachineConfig(dev, vzVMConfig); err != nil {
			return nil, err
		}
	}

	if vm.Timesync != nil && vm.Timesync.VsockPort != 0 {
		// automatically add the vsock device we'll need for communication over VsockPort
		vsockDev := VirtioVsock{
			Port:   vm.Timesync.VsockPort,
			Listen: false,
		}
		if err := vsockDev.AddToVirtualMachineConfig(vzVMConfig); err != nil {
			return nil, err
		}
	}

	valid, err := vzVMConfig.Validate()
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("Invalid virtual machine configuration")
	}

	return vzVMConfig, nil
}
