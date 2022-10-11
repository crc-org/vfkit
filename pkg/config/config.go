package config

import (
	"fmt"
	"os"

	"github.com/Code-Hex/vz/v2"
	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
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

func isKernelUncompressed(filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	buf := make([]byte, 2048)
	_, err = file.Read(buf)
	if err != nil {
		return false, err
	}
	kind, err := filetype.Match(buf)
	if err != nil {
		return false, err
	}
	// uncompressed ARM64 kernels are matched as a MS executable, which is
	// also an archive, so we need to special case it
	if kind == matchers.TypeExe {
		return true, nil
	}

	return false, nil
}

func (bootloader *Bootloader) toVzBootloader() (vz.BootLoader, error) {
	uncompressed, err := isKernelUncompressed(bootloader.vmlinuzPath)
	if err != nil {
		return nil, err
	}
	if !uncompressed {
		return nil, fmt.Errorf("kernel must be uncompressed, %s is a compressed file", bootloader.vmlinuzPath)
	}

	return vz.NewLinuxBootLoader(
		bootloader.vmlinuzPath,
		vz.WithCommandLine(bootloader.kernelCmdLine),
		vz.WithInitrd(bootloader.initrdPath),
	)
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

	vzVMConfig, err := vz.NewVirtualMachineConfiguration(vzBootloader, vm.vcpus, vm.memoryBytes)
	if err != nil {
		return nil, err
	}

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
