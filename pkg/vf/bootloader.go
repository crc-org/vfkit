package vf

import (
	"fmt"
	"os"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
)

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

func toVzLinuxBootloader(bootloader *config.LinuxBootloader) (vz.BootLoader, error) {
	if runtime.GOARCH == "arm64" {
		uncompressed, err := isKernelUncompressed(bootloader.VmlinuzPath)
		if err != nil {
			return nil, err
		}
		if !uncompressed {
			return nil, fmt.Errorf("kernel must be uncompressed, %s is a compressed file", bootloader.VmlinuzPath)
		}
	}

	return vz.NewLinuxBootLoader(
		bootloader.VmlinuzPath,
		vz.WithCommandLine(bootloader.KernelCmdLine),
		vz.WithInitrd(bootloader.InitrdPath),
	)
}

func toVzEFIBootloader(bootloader *config.EFIBootloader) (vz.BootLoader, error) {
	var efiVariableStore *vz.EFIVariableStore
	var err error

	if bootloader.CreateVariableStore {
		efiVariableStore, err = vz.NewEFIVariableStore(bootloader.EFIVariableStorePath, vz.WithCreatingEFIVariableStore())
	} else {
		efiVariableStore, err = vz.NewEFIVariableStore(bootloader.EFIVariableStorePath)
	}
	if err != nil {
		return nil, err
	}

	return vz.NewEFIBootLoader(
		vz.WithEFIVariableStore(efiVariableStore),
	)
}

func ToVzBootloader(bootloader config.Bootloader) (vz.BootLoader, error) {
	switch b := bootloader.(type) {
	case *config.LinuxBootloader:
		return toVzLinuxBootloader(b)
	case *config.EFIBootloader:
		return toVzEFIBootloader(b)
	default:
		return nil, fmt.Errorf("Unexpected bootloader type: %T", b)
	}
}
