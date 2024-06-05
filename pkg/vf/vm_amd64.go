package vf

import (
	"fmt"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
)

func NewMacPlatformConfiguration() (vz.PlatformConfiguration, error) {
	return nil, fmt.Errorf("macOS guests are only supported on ARM devices.")
}

func toVzMacOSBootloader(bootloader *config.MacOSBootloader) (vz.BootLoader, error) {
	return nil, fmt.Errorf("macOS guests are only supported on ARM devices.")
}

func newMacGraphicsDeviceConfiguration(dev *VirtioGPU) (vz.GraphicsDeviceConfiguration, error) {
	return nil, fmt.Errorf("macOS guests are only supported on ARM devices.")
}
