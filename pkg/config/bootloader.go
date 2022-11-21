package config

import (
	"fmt"
	"os"

	"github.com/Code-Hex/vz/v3"
	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
)

type Bootloader interface {
	toVzBootloader() (vz.BootLoader, error)
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
