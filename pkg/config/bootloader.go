package config

import (
	"fmt"
	"strings"

	"github.com/crc-org/vfkit/pkg/util"
)

type Bootloader interface {
	FromOptions(options []option) error
	ToCmdLine() ([]string, error)
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

// NewLinuxBootloader creates a new bootloader to start a VM with the file at
// vmlinuzPath as the kernel, kernelCmdLine as the kernel command line, and the
// file at initrdPath as the initrd. On ARM64, the kernel must be uncompressed
// otherwise the VM will fail to boot.
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

func (bootloader *LinuxBootloader) ToCmdLine() ([]string, error) {
	args := []string{}
	if bootloader.VmlinuzPath == "" {
		return nil, fmt.Errorf("Missing kernel path")
	}
	args = append(args, "--kernel", bootloader.VmlinuzPath)

	if bootloader.InitrdPath == "" {
		return nil, fmt.Errorf("Missing initrd path")
	}
	args = append(args, "--initrd", bootloader.InitrdPath)

	if bootloader.KernelCmdLine == "" {
		return nil, fmt.Errorf("Missing kernel command line")
	}
	args = append(args, "--kernel-cmdline", bootloader.KernelCmdLine)

	return args, nil
}

// NewEFIBootloader creates a new bootloader to start a VM using EFI
// efiVariableStorePath is the path to a file for EFI storage
// create is a boolean indicating if the file for the store should be created or not
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

func (bootloader *EFIBootloader) ToCmdLine() ([]string, error) {
	if bootloader.EFIVariableStorePath == "" {
		return nil, fmt.Errorf("Missing EFI store path")
	}

	builder := strings.Builder{}
	builder.WriteString("efi")
	builder.WriteString(fmt.Sprintf(",variable-store=%s", bootloader.EFIVariableStorePath))
	if bootloader.CreateVariableStore {
		builder.WriteString(",create")
	}

	return []string{"--bootloader", builder.String()}, nil
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
