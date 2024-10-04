package vf

import (
	"fmt"
	"os"

	"github.com/Code-Hex/vz/v3"
	log "github.com/sirupsen/logrus"
)

var (
	checkRosettaDirectoryShareAvailability = vz.LinuxRosettaDirectoryShareAvailability
	doInstallRosetta                       = vz.LinuxRosettaDirectoryShareInstallRosetta
)

func (dev *RosettaShare) checkRosettaAvailability() error {
	availability := checkRosettaDirectoryShareAvailability()
	switch availability {
	case vz.LinuxRosettaAvailabilityNotSupported:
		return fmt.Errorf("rosetta is not supported")
	case vz.LinuxRosettaAvailabilityNotInstalled:
		if !dev.InstallRosetta {
			return fmt.Errorf("rosetta is not installed")
		}
		log.Debugf("installing rosetta")
		if err := doInstallRosetta(); err != nil {
			if dev.IgnoreIfMissing {
				log.Info("Rosetta installation failed. Continuing without Rosetta.")
				_, err = os.Stderr.WriteString(err.Error() + "\n")
				if err != nil {
					log.Debugf("Failed to write error to stderr: %v", err)
				}
				return nil
			}
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
	if err := dev.checkRosettaAvailability(); err != nil {
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
