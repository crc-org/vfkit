package vf

import (
	"fmt"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
)

type vzVirtualMachineConfiguration struct {
	*vz.VirtualMachineConfiguration
	storageDevicesConfiguration          []vz.StorageDeviceConfiguration
	directorySharingDevicesConfiguration []vz.DirectorySharingDeviceConfiguration
	keyboardConfiguration                []vz.KeyboardConfiguration
	pointingDevicesConfiguration         []vz.PointingDeviceConfiguration
	graphicsDevicesConfiguration         []vz.GraphicsDeviceConfiguration
	networkDevicesConfiguration          []*vz.VirtioNetworkDeviceConfiguration
	entropyDevicesConfiguration          []*vz.VirtioEntropyDeviceConfiguration
	serialPortsConfiguration             []*vz.VirtioConsoleDeviceSerialPortConfiguration
	socketDevicesConfiguration           []vz.SocketDeviceConfiguration
}

func newVzVirtualMachineConfiguration(vm *config.VirtualMachine) (*vzVirtualMachineConfiguration, error) {
	vzBootloader, err := toVzBootloader(vm.Bootloader)
	if err != nil {
		return nil, err
	}

	vzVMConfig, err := vz.NewVirtualMachineConfiguration(vzBootloader, vm.Vcpus, uint64(vm.Memory.ToBytes()))
	if err != nil {
		return nil, err
	}

	return &vzVirtualMachineConfiguration{
		VirtualMachineConfiguration: vzVMConfig,
	}, nil
}

func ToVzVirtualMachineConfig(vm *config.VirtualMachine) (*vz.VirtualMachineConfiguration, error) {
	vzVMConfig, err := newVzVirtualMachineConfiguration(vm)
	if err != nil {
		return nil, err
	}

	for _, dev := range vm.Devices {
		if err := AddToVirtualMachineConfig(dev, vzVMConfig); err != nil {
			return nil, err
		}
	}
	vzVMConfig.SetStorageDevicesVirtualMachineConfiguration(vzVMConfig.storageDevicesConfiguration)
	vzVMConfig.SetDirectorySharingDevicesVirtualMachineConfiguration(vzVMConfig.directorySharingDevicesConfiguration)
	vzVMConfig.SetPointingDevicesVirtualMachineConfiguration(vzVMConfig.pointingDevicesConfiguration)
	vzVMConfig.SetKeyboardsVirtualMachineConfiguration(vzVMConfig.keyboardConfiguration)
	vzVMConfig.SetGraphicsDevicesVirtualMachineConfiguration(vzVMConfig.graphicsDevicesConfiguration)
	vzVMConfig.SetNetworkDevicesVirtualMachineConfiguration(vzVMConfig.networkDevicesConfiguration)
	vzVMConfig.SetEntropyDevicesVirtualMachineConfiguration(vzVMConfig.entropyDevicesConfiguration)
	vzVMConfig.SetSerialPortsVirtualMachineConfiguration(vzVMConfig.serialPortsConfiguration)
	// len(vzVMConfig.socketDevicesConfiguration should be 0 or 1
	// https://developer.apple.com/documentation/virtualization/vzvirtiosocketdeviceconfiguration?language=objc
	vzVMConfig.SetSocketDevicesVirtualMachineConfiguration(vzVMConfig.socketDevicesConfiguration)

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

	return vzVMConfig.VirtualMachineConfiguration, nil
}
