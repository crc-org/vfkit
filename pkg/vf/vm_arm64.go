package vf

import (
	"fmt"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
)

func NewMacPlatformConfiguration(machineIdentifierVar, hardwareModelVar, auxiliaryStorageVar string) (vz.PlatformConfiguration, error) {
	// The following string is common for the hardware model:
	// `YnBsaXN0MDDTAQIDBAUGXxAZRGF0YVJlcHJlc2VudGF0aW9uVmVyc2lvbl8QD1BsYXRmb3JtVmVyc2lvbl8QEk1pbmltdW1TdXBwb3J0ZWRPUxQAAAAAAAAAAAAAAAAAAAABEAKjBwgIEA0QAAgPKz1SY2VpawAAAAAAAAEBAAAAAAAAAAkAAAAAAAAAAAAAAAAAAABt`
	// It is a base64-encoded binary plist with this content: `{"DataRepresentationVersion":1,"MinimumSupportedOS":[13,0,0],"PlatformVersion":2}`
	hardwareModel, err := vz.NewMacHardwareModelWithDataPath(hardwareModelVar)

	if err != nil {
		return nil, fmt.Errorf("hardwareModel error: %w", err)
	}

	macAuxiliaryStorage, err := vz.NewMacAuxiliaryStorage(
		auxiliaryStorageVar,
		vz.WithCreatingMacAuxiliaryStorage(hardwareModel),
	)

	if err != nil {
		return nil, fmt.Errorf("macAuxiliaryStorage error: %w", err)
	}

	machineIdentifier, err := vz.NewMacMachineIdentifierWithDataPath(
		machineIdentifierVar,
	)

	if err != nil {
		return nil, fmt.Errorf("machineIdentifier error: %w", err)
	}

	platformConfig, err := vz.NewMacPlatformConfiguration(
		vz.WithMacAuxiliaryStorage(macAuxiliaryStorage),
		vz.WithMacHardwareModel(hardwareModel),
		vz.WithMacMachineIdentifier(machineIdentifier),
	)

	if err != nil {
		return nil, err
	}

	return platformConfig, nil
}

func toVzMacOSBootloader(_ *config.MacOSBootloader) (vz.BootLoader, error) {
	return vz.NewMacOSBootLoader()
}

func newMacGraphicsDeviceConfiguration(dev *VirtioGPU) (vz.GraphicsDeviceConfiguration, error) {
	const MacDisplayPixelsPerInch = int64(80) // Hardcoded since HiDPI scaling doesn't seem to work

	gpuDeviceConfig, err := vz.NewMacGraphicsDeviceConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize macOS graphics device: %w", err)
	}
	graphicsDisplayConfig, err := vz.NewMacGraphicsDisplayConfiguration(int64(dev.Width), int64(dev.Height), MacDisplayPixelsPerInch)

	if err != nil {
		return nil, fmt.Errorf("failed to create macOS graphics configuration: %w", err)
	}

	gpuDeviceConfig.SetDisplays(
		graphicsDisplayConfig,
	)

	return gpuDeviceConfig, nil
}
