package vf

import (
	"fmt"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
)

func NewMacPlatformConfiguration(auxiliaryStorageVar, hardwareModelVar, machineIdentifierVar string) (vz.PlatformConfiguration, error) {
	return nil, fmt.Errorf("Running macOS guests is only supported on ARM devices")
}

func toVzMacOSBootloader(_bootloader *config.MacOSBootloader) (vz.BootLoader, error) {
	return nil, fmt.Errorf("Running macOS guests is only supported on ARM devices")
}

func newMacGraphicsDeviceConfiguration(dev *VirtioGPU) (vz.GraphicsDeviceConfiguration, error) {
	return nil, fmt.Errorf("Running macOS guests is only supported on ARM devices")
}
