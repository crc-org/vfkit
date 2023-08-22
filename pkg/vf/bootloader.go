package vf

import (
	"bytes"
	"fmt"
	"os"
	"runtime"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
)

// from https://github.com/h2non/filetype/blob/cfcd7d097bc4990dc8fc86187307651ae79bf9d9/matchers/document.go#L159-L174
func compareBytes(slice, subSlice []byte, startOffset int) bool {
	sl := len(subSlice)

	if startOffset+sl > len(slice) {
		return false
	}

	s := slice[startOffset : startOffset+sl]
	return bytes.Equal(s, subSlice)
}

// patterns and offsets are coming from https://github.com/file/file/blob/master/magic/Magdir/linux
func isUncompressedArm64Kernel(buf []byte) bool {
	pattern := []byte{0x41, 0x52, 0x4d, 0x64}
	offset := 0x38

	return compareBytes(buf, pattern, offset)
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
	return isUncompressedArm64Kernel(buf), nil
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
