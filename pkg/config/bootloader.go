package config

import (
	"fmt"

	"github.com/crc-org/vfkit/pkg/util"
)

type Bootloader interface {
	FromOptions(options []option) error
}

type LinuxBootloader struct {
	VmlinuzPath   string
	KernelCmdLine string
	InitrdPath    string
}

type EFIBootloader struct {
	EFIVariableStorePath string
	// TODO: virtualization framework allow both create and overwrite
	CreateVariableStore bool
}

func NewLinuxBootloader(vmlinuzPath, kernelCmdLine, initrdPath string) *LinuxBootloader {
	return &LinuxBootloader{
		VmlinuzPath:   vmlinuzPath,
		KernelCmdLine: kernelCmdLine,
		InitrdPath:    initrdPath,
	}
}

func (bootloader *LinuxBootloader) FromOptions(options []option) error {
	for _, option := range options {
		switch option.key {
		case "kernel":
			bootloader.VmlinuzPath = option.value
		case "cmdline":
			bootloader.KernelCmdLine = util.TrimQuotes(option.value)
		case "initrd":
			bootloader.InitrdPath = option.value
		default:
			return fmt.Errorf("Unknown option for linux bootloaders: %s", option.key)
		}
	}
	return nil
}

func NewEFIBootloader(efiVariableStorePath string, createVariableStore bool) *EFIBootloader {
	return &EFIBootloader{
		EFIVariableStorePath: efiVariableStorePath,
		CreateVariableStore:  createVariableStore,
	}
}

func (bootloader *EFIBootloader) FromOptions(options []option) error {
	for _, option := range options {
		switch option.key {
		case "variable-store":
			bootloader.EFIVariableStorePath = option.value
		case "create":
			if option.value != "" {
				return fmt.Errorf("Unexpected value for EFI bootloader 'create' option: %s", option.value)
			}
			bootloader.CreateVariableStore = true
		default:
			return fmt.Errorf("Unknown option for EFI bootloaders: %s", option.key)
		}
	}
	return nil
}

func BootloaderFromCmdLine(optsStrv []string) (Bootloader, error) {
	var bootloader Bootloader

	if len(optsStrv) < 1 {
		return nil, fmt.Errorf("empty option list in --bootloader command line argument")
	}
	bootloaderType := optsStrv[0]
	switch bootloaderType {
	case "efi":
		bootloader = &EFIBootloader{}
	case "linux":
		bootloader = &LinuxBootloader{}
	default:
		return nil, fmt.Errorf("unknown bootloader type: %s", bootloaderType)
	}
	options := strvToOptions(optsStrv[1:])
	if err := bootloader.FromOptions(options); err != nil {
		return nil, err
	}
	return bootloader, nil
}
