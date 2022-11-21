package config

import (
	"fmt"
	"os"

	"github.com/crc-org/vfkit/pkg/util"

	"github.com/Code-Hex/vz/v3"
	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
)

type Bootloader interface {
	toVzBootloader() (vz.BootLoader, error)
	FromOptions(options []option) error
}

type LinuxBootloader struct {
	vmlinuzPath   string
	kernelCmdLine string
	initrdPath    string
}

type EFIBootloader struct {
	efiVariableStorePath string
	createVariableStore  bool
}

func NewLinuxBootloader(vmlinuzPath, kernelCmdLine, initrdPath string) *LinuxBootloader {
	return &LinuxBootloader{
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

func (bootloader *LinuxBootloader) toVzBootloader() (vz.BootLoader, error) {
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

func (bootloader *LinuxBootloader) FromOptions(options []option) error {
	for _, option := range options {
		switch option.key {
		case "kernel":
			bootloader.vmlinuzPath = option.value
		case "cmdline":
			bootloader.kernelCmdLine = util.TrimQuotes(option.value)
		case "initrd":
			bootloader.initrdPath = option.value
		default:
			return fmt.Errorf("Unknown option for linux bootloaders: %s", option.key)
		}
	}
	return nil
}

func NewEFIBootloader(efiVariableStorePath string, createVariableStore bool) *EFIBootloader {
	return &EFIBootloader{
		efiVariableStorePath: efiVariableStorePath,
		createVariableStore:  createVariableStore,
	}
}

func (bootloader *EFIBootloader) toVzBootloader() (vz.BootLoader, error) {
	var efiVariableStore *vz.EFIVariableStore
	var err error

	if bootloader.createVariableStore {
		efiVariableStore, err = vz.NewEFIVariableStore(bootloader.efiVariableStorePath, vz.WithCreatingEFIVariableStore())
	} else {
		efiVariableStore, err = vz.NewEFIVariableStore(bootloader.efiVariableStorePath)
	}
	if err != nil {
		return nil, err
	}

	return vz.NewEFIBootLoader(
		vz.WithEFIVariableStore(efiVariableStore),
	)
}

func (bootloader *EFIBootloader) FromOptions(options []option) error {
	for _, option := range options {
		switch option.key {
		case "variable-store":
			bootloader.efiVariableStorePath = option.value
		case "create":
			if option.value != "" {
				return fmt.Errorf("Unexpected value for EFI bootloader 'create' option: %s", option.value)
			}
			bootloader.createVariableStore = true
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
