package vf

import (
	"fmt"
	"path/filepath"

	"github.com/crc-org/vfkit/pkg/config"

	"github.com/Code-Hex/vz/v3"
	log "github.com/sirupsen/logrus"
)

type VirtioBlk config.VirtioBlk
type VirtioFs config.VirtioFs
type VirtioNet config.VirtioNet
type VirtioRng config.VirtioRng
type VirtioSerial config.VirtioSerial
type VirtioVsock config.VirtioVsock

func (dev *VirtioBlk) AddToVirtualMachineConfig(vmConfig *vz.VirtualMachineConfiguration) error {
	if dev.ImagePath == "" {
		return fmt.Errorf("missing mandatory 'path' option for virtio-blk device")
	}
	log.Infof("Adding virtio-blk device (imagePath: %s)", dev.ImagePath)
	diskImageAttachment, err := vz.NewDiskImageStorageDeviceAttachment(
		dev.ImagePath,
		false,
	)
	if err != nil {
		return err
	}
	storageDeviceConfig, err := vz.NewVirtioBlockDeviceConfiguration(diskImageAttachment)
	if err != nil {
		return err
	}
	vmConfig.SetStorageDevicesVirtualMachineConfiguration([]vz.StorageDeviceConfiguration{
		storageDeviceConfig,
	})
	return nil
}
func (dev *VirtioFs) AddToVirtualMachineConfig(vmConfig *vz.VirtualMachineConfiguration) error {
	log.Infof("Adding virtio-fs device")
	if dev.SharedDir == "" {
		return fmt.Errorf("missing mandatory 'sharedDir' option for virtio-fs device")
	}
	var mountTag string
	if dev.MountTag != "" {
		mountTag = dev.MountTag
	} else {
		mountTag = filepath.Base(dev.SharedDir)
	}

	sharedDir, err := vz.NewSharedDirectory(dev.SharedDir, false)
	if err != nil {
		return err
	}
	sharedDirConfig, err := vz.NewSingleDirectoryShare(sharedDir)
	if err != nil {
		return err
	}
	fileSystemDeviceConfig, err := vz.NewVirtioFileSystemDeviceConfiguration(mountTag)
	if err != nil {
		return err
	}
	fileSystemDeviceConfig.SetDirectoryShare(sharedDirConfig)
	vmConfig.SetDirectorySharingDevicesVirtualMachineConfiguration([]vz.DirectorySharingDeviceConfiguration{
		fileSystemDeviceConfig,
	})
	return nil
}

func (dev *VirtioNet) AddToVirtualMachineConfig(vmConfig *vz.VirtualMachineConfiguration) error {
	var (
		mac *vz.MACAddress
		err error
	)

	if !dev.Nat {
		return fmt.Errorf("NAT is the only supported networking mode")
	}

	log.Infof("Adding virtio-net device (nat: %t macAddress: [%s])", dev.Nat, dev.MacAddress)

	if len(dev.MacAddress) == 0 {
		mac, err = vz.NewRandomLocallyAdministeredMACAddress()
	} else {
		mac, err = vz.NewMACAddress(dev.MacAddress)
	}
	if err != nil {
		return err
	}
	natAttachment, err := vz.NewNATNetworkDeviceAttachment()
	if err != nil {
		return err
	}
	networkConfig, err := vz.NewVirtioNetworkDeviceConfiguration(natAttachment)
	if err != nil {
		return err
	}
	networkConfig.SetMACAddress(mac)
	vmConfig.SetNetworkDevicesVirtualMachineConfiguration([]*vz.VirtioNetworkDeviceConfiguration{
		networkConfig,
	})

	return nil
}

func (dev *VirtioRng) AddToVirtualMachineConfig(vmConfig *vz.VirtualMachineConfiguration) error {
	log.Infof("Adding virtio-rng device")
	entropyConfig, err := vz.NewVirtioEntropyDeviceConfiguration()
	if err != nil {
		return err
	}
	vmConfig.SetEntropyDevicesVirtualMachineConfiguration([]*vz.VirtioEntropyDeviceConfiguration{
		entropyConfig,
	})

	return nil
}

func (dev *VirtioSerial) AddToVirtualMachineConfig(vmConfig *vz.VirtualMachineConfiguration) error {
	if dev.LogFile == "" {
		return fmt.Errorf("missing mandatory 'logFile' option for virtio-serial device")
	}
	log.Infof("Adding virtio-serial device (logFile: %s)", dev.LogFile)

	//serialPortAttachment := vz.NewFileHandleSerialPortAttachment(os.Stdin, tty)
	serialPortAttachment, err := vz.NewFileSerialPortAttachment(dev.LogFile, false)
	if err != nil {
		return err
	}
	consoleConfig, err := vz.NewVirtioConsoleDeviceSerialPortConfiguration(serialPortAttachment)
	if err != nil {
		return err
	}
	vmConfig.SetSerialPortsVirtualMachineConfiguration([]*vz.VirtioConsoleDeviceSerialPortConfiguration{
		consoleConfig,
	})

	return nil
}

func (dev *VirtioVsock) AddToVirtualMachineConfig(vmConfig *vz.VirtualMachineConfiguration) error {
	if len(vmConfig.SocketDevices()) != 0 {
		log.Debugf("virtio-vsock device already present, not adding a second one")
		return nil
	}
	log.Infof("Adding virtio-vsock device")
	vzdev, err := vz.NewVirtioSocketDeviceConfiguration()
	if err != nil {
		return err
	}
	vmConfig.SetSocketDevicesVirtualMachineConfiguration([]vz.SocketDeviceConfiguration{vzdev})

	return nil
}

func AddToVirtualMachineConfig(dev config.VirtioDevice, vmConfig *vz.VirtualMachineConfiguration) error {
	switch d := dev.(type) {
	case *config.VirtioBlk:
		return (*VirtioBlk)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioFs:
		return (*VirtioFs)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioNet:
		return (*VirtioNet)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioRng:
		return (*VirtioRng)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioSerial:
		return (*VirtioSerial)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioVsock:
		return (*VirtioVsock)(d).AddToVirtualMachineConfig(vmConfig)
	default:
		return fmt.Errorf("Unexpected virtio device type: %T", d)
	}
}
