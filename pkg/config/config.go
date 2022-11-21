package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Code-Hex/vz/v3"
)

type VirtualMachine struct {
	vcpus       uint
	memoryBytes uint64
	bootloader  Bootloader
	devices     []VirtioDevice
	timesync    *TimeSync
}

type TimeSync struct {
	vsockPort uint
}

func (ts *TimeSync) VsockPort() uint {
	return ts.vsockPort
}

func NewVirtualMachine(vcpus uint, memoryBytes uint64, bootloader Bootloader) *VirtualMachine {
	return &VirtualMachine{
		vcpus:       vcpus,
		memoryBytes: memoryBytes,
		bootloader:  bootloader,
	}
}

func (vm *VirtualMachine) AddTimeSyncFromCmdLine(cmdlineOpts string) error {
	if cmdlineOpts == "" {
		return nil
	}
	timesync, err := timesyncFromCmdLine(cmdlineOpts)
	if err != nil {
		return err
	}
	vm.timesync = timesync

	return nil
}

func (vm *VirtualMachine) AddDevicesFromCmdLine(cmdlineOpts []string) error {
	for _, deviceOpts := range cmdlineOpts {
		dev, err := deviceFromCmdLine(deviceOpts)
		if err != nil {
			return err
		}
		vm.devices = append(vm.devices, dev)
	}
	return nil
}

func (vm *VirtualMachine) ToVzVirtualMachineConfig() (*vz.VirtualMachineConfiguration, error) {
	vzBootloader, err := vm.bootloader.toVzBootloader()
	if err != nil {
		return nil, err
	}

	vzVMConfig, err := vz.NewVirtualMachineConfiguration(vzBootloader, vm.vcpus, vm.memoryBytes)
	if err != nil {
		return nil, err
	}

	for _, dev := range vm.devices {
		if err := dev.AddToVirtualMachineConfig(vzVMConfig); err != nil {
			return nil, err
		}
	}

	if vm.timesync != nil && vm.timesync.VsockPort() != 0 {
		// automatically add the vsock device we'll need for communication over VsockPort()
		vsockDev := VirtioVsock{
			Port:   vm.timesync.VsockPort(),
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

func (vm *VirtualMachine) TimeSync() *TimeSync {
	return vm.timesync
}

func (vm *VirtualMachine) VirtioVsockDevices() []*VirtioVsock {
	vsockDevs := []*VirtioVsock{}
	for _, dev := range vm.devices {
		if vsockDev, isVirtioVsock := dev.(*VirtioVsock); isVirtioVsock {
			vsockDevs = append(vsockDevs, vsockDev)
		}
	}

	return vsockDevs
}

func timesyncFromCmdLine(optsStr string) (*TimeSync, error) {
	var timesync TimeSync

	optsStrv := strings.Split(optsStr, ",")
	options := strvToOptions(optsStrv)

	for _, option := range options {
		switch option.key {
		case "vsockPort":
			vsockPort, err := strconv.ParseUint(option.value, 10, 64)
			if err != nil {
				return nil, err
			}
			timesync.vsockPort = uint(vsockPort)
		default:
			return nil, fmt.Errorf("Unknown option for timesync parameter: %s", option.key)
		}
	}

	if timesync.vsockPort == 0 {
		return nil, fmt.Errorf("Missing 'vsockPort' option for timesync parameter")
	}

	return &timesync, nil
}
