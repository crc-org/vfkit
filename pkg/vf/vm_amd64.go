package vf

import (
	"fmt"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
)

func NewMacPlatformConfiguration(_, _, _ string) (vz.PlatformConfiguration, error) {
	return nil, fmt.Errorf("running macOS guests is only supported on ARM devices")
}

func toVzMacOSBootloader(_ *config.MacOSBootloader) (vz.BootLoader, error) {
	return nil, fmt.Errorf("running macOS guests is only supported on ARM devices")
}

func newMacGraphicsDeviceConfiguration(_ *VirtioGPU) (vz.GraphicsDeviceConfiguration, error) {
	return nil, fmt.Errorf("running macOS guests is only supported on ARM devices")
}
