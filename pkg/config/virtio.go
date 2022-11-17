package config

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Code-Hex/vz/v3"
	log "github.com/sirupsen/logrus"
)

type VirtioDevice interface {
	FromOptions([]option) error
	AddToVirtualMachineConfig(*vz.VirtualMachineConfiguration) error
}

type VirtioVsock struct {
	Port      uint
	SocketURL string
	Listen    bool
}

type VirtioBlk struct {
	ImagePath string
}

type VirtioRng struct {
}

// TODO: Add BridgedNetwork support
// https://github.com/Code-Hex/vz/blob/d70a0533bf8ed0fa9ab22fa4d4ca554b7c3f3ce5/network.go#L81-L82

// TODO: Add FileHandleNetwork support
// https://github.com/Code-Hex/vz/blob/d70a0533bf8ed0fa9ab22fa4d4ca554b7c3f3ce5/network.go#L109-L112
type VirtioNet struct {
	Nat        bool
	MacAddress net.HardwareAddr
}

type VirtioSerial struct {
	LogFile string
}

// TODO: Add VirtioBalloon
// https://github.com/Code-Hex/vz/blob/master/memory_balloon.go

type option struct {
	key   string
	value string
}

func strToOption(str string) option {
	splitStr := strings.SplitN(str, "=", 2)

	opt := option{
		key: splitStr[0],
	}
	if len(splitStr) > 1 {
		opt.value = splitStr[1]
	}

	return opt
}

func strvToOptions(opts []string) []option {
	parsedOpts := []option{}
	for _, opt := range opts {
		if len(opt) == 0 {
			continue
		}
		parsedOpts = append(parsedOpts, strToOption(opt))
	}

	return parsedOpts
}

func deviceFromCmdLine(deviceOpts string) (VirtioDevice, error) {
	opts := strings.Split(deviceOpts, ",")
	if len(opts) == 0 {
		return nil, fmt.Errorf("empty option list in command line argument")
	}
	var dev VirtioDevice
	switch opts[0] {
	case "virtio-blk":
		dev = &VirtioBlk{}
	case "virtio-fs":
		dev = &VirtioFs{}
	case "virtio-net":
		dev = &VirtioNet{}
	case "virtio-rng":
		dev = &VirtioRng{}
	case "virtio-serial":
		dev = &VirtioSerial{}
	case "virtio-vsock":
		dev = &VirtioVsock{}
	default:
		return nil, fmt.Errorf("unknown device type: %s", opts[0])
	}

	parsedOpts := strvToOptions(opts[1:])
	if err := dev.FromOptions(parsedOpts); err != nil {
		return nil, err
	}

	return dev, nil
}

func (dev *VirtioSerial) FromOptions(options []option) error {
	for _, option := range options {
		switch option.key {
		case "logFilePath":
			dev.LogFile = option.value
		default:
			return fmt.Errorf("Unknown option for virtio-serial devices: %s", option.key)
		}
	}
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

func (dev *VirtioNet) FromOptions(options []option) error {
	for _, option := range options {
		switch option.key {
		case "nat":
			if option.value != "" {
				return fmt.Errorf("Unexpected value for virtio-net 'nat' option: %s", option.value)
			}
			dev.Nat = true
		case "mac":
			macAddress, err := net.ParseMAC(option.value)
			if err != nil {
				return err
			}
			dev.MacAddress = macAddress
		default:
			return fmt.Errorf("Unknown option for virtio-net devices: %s", option.key)
		}
	}
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

func (dev *VirtioRng) FromOptions(options []option) error {
	if len(options) != 0 {
		return fmt.Errorf("Unknown options for virtio-rng devices: %s", options)
	}
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

func (dev *VirtioBlk) FromOptions(options []option) error {
	for _, option := range options {
		switch option.key {
		case "path":
			dev.ImagePath = option.value
		default:
			return fmt.Errorf("Unknown option for virtio-blk devices: %s", option.key)
		}
	}
	return nil
}

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

func (dev *VirtioVsock) FromOptions(options []option) error {
	// default to listen for backwards compatibliity
	dev.Listen = true
	for _, option := range options {
		switch option.key {
		case "socketURL":
			dev.SocketURL = option.value
		case "port":
			port, err := strconv.Atoi(option.value)
			if err != nil {
				return err
			}
			dev.Port = uint(port)
		case "listen":
			dev.Listen = true
		case "connect":
			dev.Listen = false
		default:
			return fmt.Errorf("Unknown option for virtio-vsock devices: %s", option.key)
		}
	}
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

type VirtioFs struct {
	SharedDir string
	MountTag  string
}

func (dev *VirtioFs) FromOptions(options []option) error {
	for _, option := range options {
		switch option.key {
		case "sharedDir":
			dev.SharedDir = option.value
		case "mountTag":
			dev.MountTag = option.value
		default:
			return fmt.Errorf("Unknown option for virtio-fs devices: %s", option.key)
		}
	}
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
