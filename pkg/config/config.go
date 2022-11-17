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

type VMComponent interface {
	FromOptions([]option) error
	ToCmdLine() ([]string, error)
}

// NewVirtualMachine creates a new VirtualMachine instance. The virtual machine
// will use vcpus virtual CPUs and it will be allocated memoryBytes bytes of
// RAM. bootloader specifies which kernel/initrd/kernel args it will be using.
func NewVirtualMachine(vcpus uint, memoryBytes uint64, bootloader Bootloader) *VirtualMachine {
	return &VirtualMachine{
		Vcpus:       vcpus,
		MemoryBytes: memoryBytes,
		Bootloader:  bootloader,
	}
}

// ToCmdLine generates a list of arguments for use with the [os/exec] package.
// These arguments will start a virtual machine with the devices/bootloader/...
// described by vm If the virtual machine configuration described by vm is
// invalid, an error will be returned.
func (vm *VirtualMachine) ToCmdLine() ([]string, error) {
	// TODO: missing binary name/path
	args := []string{}

	if vm.Vcpus != 0 {
		args = append(args, "--cpus", strconv.FormatUint(uint64(vm.Vcpus), 10))
	}
	if vm.MemoryBytes != 0 {
		args = append(args, "--memory", strconv.FormatUint(vm.MemoryBytes, 10))
	}

	if vm.Bootloader == nil {
		return nil, fmt.Errorf("missing bootloader configuration")
	}
	bootloaderArgs, err := vm.Bootloader.ToCmdLine()
	if err != nil {
		return nil, err
	}
	args = append(args, bootloaderArgs...)

	for _, dev := range vm.Devices {
		devArgs, err := dev.ToCmdLine()
		if err != nil {
			return nil, err
		}
		args = append(args, devArgs...)
	}

	return args, nil
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

func (vm *VirtualMachine) VirtioVsockDevices() []*VirtioVsock {
	vsockDevs := []*VirtioVsock{}
	for _, dev := range vm.Devices {
		if vsockDev, isVirtioVsock := dev.(*VirtioVsock); isVirtioVsock {
			vsockDevs = append(vsockDevs, vsockDev)
		}
	}

	return vsockDevs
}

// AddDevice adds a dev to vm. This device can be created with one of the
// VirtioXXXNew methods.
func (vm *VirtualMachine) AddDevice(dev VirtioDevice) error {
	vm.Devices = append(vm.Devices, dev)

	return nil
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

func (vm *VirtualMachine) TimeSync() *TimeSync {
	return vm.Timesync
}

func TimeSyncNew(vsockPort uint) (VMComponent, error) {
	return &TimeSync{
		VsockPort: vsockPort,
	}, nil
}

func (ts *TimeSync) ToCmdLine() ([]string, error) {
	args := []string{}
	if ts.VsockPort != 0 {
		args = append(args, fmt.Sprintf("vsockPort=%d", ts.VsockPort))
	}
	return []string{"--timesync", strings.Join(args, ",")}, nil
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
