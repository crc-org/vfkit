package vf

import (
	"fmt"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
)

type VirtualMachineConfiguration struct {
	*vz.VirtualMachineConfiguration                             // wrapper for Objective-C type
	config                               *config.VirtualMachine // go-friendly virtual machine configuration definition
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

func NewVirtualMachineConfiguration(vmConfig *config.VirtualMachine) (*VirtualMachineConfiguration, error) {
	vzBootloader, err := toVzBootloader(vmConfig.Bootloader)
	if err != nil {
		return nil, err
	}

	vzVMConfig, err := vz.NewVirtualMachineConfiguration(vzBootloader, vmConfig.Vcpus, uint64(vmConfig.Memory.ToBytes()))
	if err != nil {
		return nil, err
	}

	return &VirtualMachineConfiguration{
		VirtualMachineConfiguration: vzVMConfig,
		config:                      vmConfig,
	}, nil
}

func (cfg *VirtualMachineConfiguration) ToVz() (*vz.VirtualMachineConfiguration, error) {
	for _, dev := range cfg.config.Devices {
		if err := AddToVirtualMachineConfig(cfg, dev); err != nil {
			return nil, err
		}
	}
	cfg.SetStorageDevicesVirtualMachineConfiguration(cfg.storageDevicesConfiguration)
	cfg.SetDirectorySharingDevicesVirtualMachineConfiguration(cfg.directorySharingDevicesConfiguration)
	cfg.SetPointingDevicesVirtualMachineConfiguration(cfg.pointingDevicesConfiguration)
	cfg.SetKeyboardsVirtualMachineConfiguration(cfg.keyboardConfiguration)
	cfg.SetGraphicsDevicesVirtualMachineConfiguration(cfg.graphicsDevicesConfiguration)
	cfg.SetNetworkDevicesVirtualMachineConfiguration(cfg.networkDevicesConfiguration)
	cfg.SetEntropyDevicesVirtualMachineConfiguration(cfg.entropyDevicesConfiguration)
	cfg.SetSerialPortsVirtualMachineConfiguration(cfg.serialPortsConfiguration)
	// len(cfg.socketDevicesConfiguration should be 0 or 1
	// https://developer.apple.com/documentation/virtualization/vzvirtiosocketdeviceconfiguration?language=objc
	cfg.SetSocketDevicesVirtualMachineConfiguration(cfg.socketDevicesConfiguration)

	if cfg.config.Timesync != nil && cfg.config.Timesync.VsockPort != 0 {
		// automatically add the vsock device we'll need for communication over VsockPort
		vsockDev := VirtioVsock{
			Port:   cfg.config.Timesync.VsockPort,
			Listen: false,
		}
		if err := vsockDev.AddToVirtualMachineConfig(cfg); err != nil {
			return nil, err
		}
	}

	valid, err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("Invalid virtual machine configuration")
	}

	return cfg.VirtualMachineConfiguration, nil
}
