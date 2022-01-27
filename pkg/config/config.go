package config

import (
	"fmt"

	"github.com/Code-Hex/vz"
)

type Bootloader struct {
	vmlinuzPath   string
	kernelCmdLine string
	initrdPath    string
}

type VirtualMachine struct {
	vcpus       uint
	memoryBytes uint64
	bootloader  *Bootloader
	devices     []VirtioDevice
}

func NewBootloader(vmlinuzPath, kernelCmdLine, initrdPath string) *Bootloader {
	return &Bootloader{
		vmlinuzPath:   vmlinuzPath,
		kernelCmdLine: kernelCmdLine,
		initrdPath:    initrdPath,
	}
}

func (bootloader *Bootloader) toVzBootloader() (vz.BootLoader, error) {
	return vz.NewLinuxBootLoader(
		bootloader.vmlinuzPath,
		vz.WithCommandLine(bootloader.kernelCmdLine),
		vz.WithInitrd(bootloader.initrdPath),
	), nil
}

func NewVirtualMachine(vcpus uint, memoryBytes uint64, bootloader *Bootloader) *VirtualMachine {
	return &VirtualMachine{
		vcpus:       vcpus,
		memoryBytes: memoryBytes,
		bootloader:  bootloader,
	}
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

	vzVMConfig := vz.NewVirtualMachineConfiguration(vzBootloader, vm.vcpus, vm.memoryBytes)

	for _, dev := range vm.devices {
		if err := dev.AddToVirtualMachineConfig(vzVMConfig); err != nil {
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

func (vm *VirtualMachine) VirtioVsockDevices() []*VirtioVsock {
	vsockDevs := []*VirtioVsock{}
	for _, dev := range vm.devices {
		if vsockDev, isVirtioVsock := dev.(*VirtioVsock); isVirtioVsock {
			vsockDevs = append(vsockDevs, vsockDev)
		}
	}

	return vsockDevs
}
