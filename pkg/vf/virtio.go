package vf

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"github.com/crc-org/vfkit/pkg/config"

	"github.com/Code-Hex/vz/v3"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

type VirtioBlk config.VirtioBlk
type VirtioFs config.VirtioFs
type VirtioNet config.VirtioNet
type VirtioRng config.VirtioRng
type VirtioSerial config.VirtioSerial
type VirtioVsock config.VirtioVsock

func (dev *VirtioBlk) toVz() (vz.StorageDeviceConfiguration, error) {
	var storageConfig StorageConfig = StorageConfig(dev.StorageConfig)
	attachment, err := storageConfig.toVz()
	if err != nil {
		return nil, err
	}
	devConfig, err := vz.NewVirtioBlockDeviceConfiguration(attachment)
	if err != nil {
		return nil, err
	}

	if dev.DeviceIdentifier != "" {
		err := devConfig.SetBlockDeviceIdentifier(dev.DeviceIdentifier)
		if err != nil {
			return nil, err
		}
	}

	return devConfig, nil
}

func (dev *VirtioBlk) AddToVirtualMachineConfig(vmConfig *vzVirtualMachineConfiguration) error {
	storageDeviceConfig, err := dev.toVz()
	if err != nil {
		return err
	}
	log.Infof("Adding virtio-blk device (imagePath: %s)", dev.ImagePath)
	vmConfig.storageDeviceConfiguration = append(vmConfig.storageDeviceConfiguration, storageDeviceConfig)

	return nil
}

func (dev *VirtioFs) AddToVirtualMachineConfig(vmConfig *vzVirtualMachineConfiguration) error {
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

func (dev *VirtioNet) connectUnixPath() error {
	conn, err := net.Dial("unix", dev.UnixSocketPath)
	if err != nil {
		return err
	}
	fd, err := conn.(*net.UnixConn).File()
	if err != nil {
		return err
	}

	dev.Socket = fd
	dev.UnixSocketPath = ""
	return nil
}

func (dev *VirtioNet) AddToVirtualMachineConfig(vmConfig *vzVirtualMachineConfiguration) error {
	var (
		mac *vz.MACAddress
		err error
	)

	log.Infof("Adding virtio-net device (nat: %t macAddress: [%s])", dev.Nat, dev.MacAddress)
	if dev.Socket != nil {
		log.Infof("Using fd %d", dev.Socket.Fd())
	}
	if dev.UnixSocketPath != "" {
		log.Infof("Using unix socket %s", dev.UnixSocketPath)
		if err := dev.connectUnixPath(); err != nil {
			return err
		}
	}

	if len(dev.MacAddress) == 0 {
		mac, err = vz.NewRandomLocallyAdministeredMACAddress()
	} else {
		mac, err = vz.NewMACAddress(dev.MacAddress)
	}
	if err != nil {
		return err
	}
	var attachment vz.NetworkDeviceAttachment
	if dev.Socket != nil {
		attachment, err = vz.NewFileHandleNetworkDeviceAttachment(dev.Socket)
	} else {
		attachment, err = vz.NewNATNetworkDeviceAttachment()
	}
	if err != nil {
		return err
	}
	networkConfig, err := vz.NewVirtioNetworkDeviceConfiguration(attachment)
	if err != nil {
		return err
	}
	networkConfig.SetMACAddress(mac)
	vmConfig.SetNetworkDevicesVirtualMachineConfiguration([]*vz.VirtioNetworkDeviceConfiguration{
		networkConfig,
	})

	return nil
}

func (dev *VirtioRng) AddToVirtualMachineConfig(vmConfig *vzVirtualMachineConfiguration) error {
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

// https://developer.apple.com/documentation/virtualization/running_linux_in_a_virtual_machine?language=objc#:~:text=Configure%20the%20Serial%20Port%20Device%20for%20Standard%20In%20and%20Out
func setRawMode(f *os.File) {
	// Get settings for terminal
	attr, _ := unix.IoctlGetTermios(int(f.Fd()), unix.TIOCGETA)

	// Put stdin into raw mode, disabling local echo, input canonicalization,
	// and CR-NL mapping.
	attr.Iflag &^= syscall.ICRNL
	attr.Lflag &^= syscall.ICANON | syscall.ECHO

	// Set minimum characters when reading = 1 char
	attr.Cc[syscall.VMIN] = 1

	// set timeout when reading as non-canonical mode
	attr.Cc[syscall.VTIME] = 0

	// reflects the changed settings
	unix.IoctlSetTermios(int(f.Fd()), unix.TIOCSETA, attr)
}

func (dev *VirtioSerial) AddToVirtualMachineConfig(vmConfig *vzVirtualMachineConfiguration) error {
	if dev.LogFile != "" {
		log.Infof("Adding virtio-serial device (logFile: %s)", dev.LogFile)
	}
	if dev.UsesStdio {
		log.Infof("Adding stdio console")
	}

	var serialPortAttachment vz.SerialPortAttachment
	var err error
	if dev.UsesStdio {
		setRawMode(os.Stdin)
		serialPortAttachment, err = vz.NewFileHandleSerialPortAttachment(os.Stdin, os.Stdout)
	} else {
		serialPortAttachment, err = vz.NewFileSerialPortAttachment(dev.LogFile, false)
	}
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

func (dev *VirtioVsock) AddToVirtualMachineConfig(vmConfig *vzVirtualMachineConfiguration) error {
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

func AddToVirtualMachineConfig(dev config.VirtioDevice, vmConfig *vzVirtualMachineConfiguration) error {
	switch d := dev.(type) {
	case *config.USBMassStorage:
		return (*USBMassStorage)(d).AddToVirtualMachineConfig(vmConfig)
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

func (config *StorageConfig) toVz() (vz.StorageDeviceAttachment, error) {
	if config.ImagePath == "" {
		return nil, fmt.Errorf("missing mandatory 'path' option for %s device", config.DevName)
	}
	return vz.NewDiskImageStorageDeviceAttachment(config.ImagePath, config.ReadOnly)
}

func (dev *USBMassStorage) toVz() (vz.StorageDeviceConfiguration, error) {
	var storageConfig StorageConfig = StorageConfig(dev.StorageConfig)
	attachment, err := storageConfig.toVz()
	if err != nil {
		return nil, err
	}
	return vz.NewUSBMassStorageDeviceConfiguration(attachment)
}

func (dev *USBMassStorage) AddToVirtualMachineConfig(vmConfig *vzVirtualMachineConfiguration) error {
	storageDeviceConfig, err := dev.toVz()
	if err != nil {
		return err
	}
	log.Infof("Adding USB mass storage device (imagePath: %s)", dev.ImagePath)
	vmConfig.storageDeviceConfiguration = append(vmConfig.storageDeviceConfiguration, storageDeviceConfig)

	return nil
}

type StorageConfig config.StorageConfig

type USBMassStorage config.USBMassStorage
