package config

import (
	"fmt"
	"strconv"
	"strings"
)

type VirtualMachine struct {
	Vcpus       uint
	MemoryBytes uint64
	Bootloader  Bootloader
	Devices     []VirtioDevice
	Timesync    *TimeSync
}

type TimeSync struct {
	VsockPort uint
}

func NewVirtualMachine(vcpus uint, memoryBytes uint64, bootloader Bootloader) *VirtualMachine {
	return &VirtualMachine{
		Vcpus:       vcpus,
		MemoryBytes: memoryBytes,
		Bootloader:  bootloader,
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
	vm.Timesync = timesync

	return nil
}

func (vm *VirtualMachine) AddDevicesFromCmdLine(cmdlineOpts []string) error {
	for _, deviceOpts := range cmdlineOpts {
		dev, err := deviceFromCmdLine(deviceOpts)
		if err != nil {
			return err
		}
		vm.Devices = append(vm.Devices, dev)
	}
	return nil
}

func (vm *VirtualMachine) TimeSync() *TimeSync {
	return vm.Timesync
}

func (vm *VirtualMachine) VirtioVsockDevices() []*VirtioVsock {
	vsockDevs := []*VirtioVsock{}
	for _, dev := range vm.Devices {
		if vsockDev, isVirtioVsock := dev.(*VirtioVsock); isVirtioVsock {
			vsockDevs = append(vsockDevs, vsockDev)
		}
	}

	return vsockDevs
}

func (ts *TimeSync) FromOptions(options []option) error {
	for _, option := range options {
		switch option.key {
		case "vsockPort":
			vsockPort, err := strconv.ParseUint(option.value, 10, 64)
			if err != nil {
				return err
			}
			ts.VsockPort = uint(vsockPort)
		default:
			return fmt.Errorf("Unknown option for timesync parameter: %s", option.key)
		}
	}

	if ts.VsockPort == 0 {
		return fmt.Errorf("Missing 'vsockPort' option for timesync parameter")
	}

	return nil
}

func timesyncFromCmdLine(optsStr string) (*TimeSync, error) {
	var timesync TimeSync

	optsStrv := strings.Split(optsStr, ",")
	options := strvToOptions(optsStrv)

	if err := timesync.FromOptions(options); err != nil {
		return nil, err
	}

	return &timesync, nil
}
