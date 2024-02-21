package vf

import (
	"fmt"

	"github.com/Code-Hex/vz/v3"
	log "github.com/sirupsen/logrus"
)

func checkRosettaAvailability(install bool) error {
	availability := vz.LinuxRosettaDirectoryShareAvailability()
	switch availability {
	case vz.LinuxRosettaAvailabilityNotSupported:
		return fmt.Errorf("rosetta is not supported")
	case vz.LinuxRosettaAvailabilityNotInstalled:
		if !install {
			return fmt.Errorf("rosetta is not installed")
		}
		log.Debugf("installing rosetta")
		if err := vz.LinuxRosettaDirectoryShareInstallRosetta(); err != nil {
			return fmt.Errorf("failed to install rosetta: %w", err)
		}
		log.Debugf("rosetta installed")
	case vz.LinuxRosettaAvailabilityInstalled:
		// nothing to do
	}

	return nil
}

func (dev *RosettaShare) toVz() (vz.DirectorySharingDeviceConfiguration, error) {
	if dev.MountTag == "" {
		return nil, fmt.Errorf("missing mandatory 'mountTage' option for rosetta share")
	}
	if err := checkRosettaAvailability(dev.InstallRosetta); err != nil {
		return nil, err
	}

	rosettaShare, err := vz.NewLinuxRosettaDirectoryShare()
	if err != nil {
		return nil, fmt.Errorf("failed to create a new rosetta directory share: %w", err)
	}
	config, err := vz.NewVirtioFileSystemDeviceConfiguration(dev.MountTag)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new virtio file system configuration for rosetta: %w", err)
	}

	config.SetDirectoryShare(rosettaShare)

	return config, nil
}

func (dev *RosettaShare) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	fileSystemDeviceConfig, err := dev.toVz()
	if err != nil {
		return err
	}
	log.Infof("Adding virtio-fs device")
	vmConfig.directorySharingDevicesConfiguration = append(vmConfig.directorySharingDevicesConfiguration, fileSystemDeviceConfig)
	return nil
}
