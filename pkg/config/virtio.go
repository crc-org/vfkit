package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type VirtioDevice VMComponent

type VirtioVsock struct {
	Port      uint
	SocketURL string
	Listen    bool
}

type VirtioBlk struct {
	ImagePath string
}

type VirtioFs struct {
	SharedDir string
	MountTag  string
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

// VirtioSerialNew creates a new serial device for the virtual machine. The
// output the virtual machine sent to the serial port will be written to the
// file at logFilePath.
func VirtioSerialNew(logFilePath string) (VirtioDevice, error) {
	return &VirtioSerial{
		LogFile: logFilePath,
	}, nil
}

func (dev *VirtioSerial) ToCmdLine() ([]string, error) {
	if dev.LogFile == "" {
		return nil, fmt.Errorf("virtio-serial needs the path to the log file")
	}
	return []string{"--device", fmt.Sprintf("virtio-serial,logFilePath=%s", dev.LogFile)}, nil
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

// VirtioNetNew creates a new network device for the virtual machine. It will
// use macAddress as its MAC address.
func VirtioNetNew(macAddress string) (VirtioDevice, error) {
	var hwAddr net.HardwareAddr

	if macAddress != "" {
		var err error
		if hwAddr, err = net.ParseMAC(macAddress); err != nil {
			return nil, err
		}
	}
	return &VirtioNet{
		Nat:        true,
		MacAddress: hwAddr,
	}, nil
}

func (dev *VirtioNet) ToCmdLine() ([]string, error) {
	if !dev.Nat {
		return nil, fmt.Errorf("virtio-net only support 'nat' networking")
	}
	builder := strings.Builder{}
	builder.WriteString("virtio-net")
	builder.WriteString(",nat")
	if len(dev.MacAddress) != 0 {
		builder.WriteString(fmt.Sprintf(",mac=%s", dev.MacAddress))
	}

	return []string{"--device", builder.String()}, nil
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

// VirtioRngNew creates a new random number generator device to feed entropy
// into the virtual machine.
func VirtioRngNew() (VirtioDevice, error) {
	return &VirtioRng{}, nil
}

func (dev *VirtioRng) ToCmdLine() ([]string, error) {
	return []string{"--device", "virtio-rng"}, nil
}

func (dev *VirtioRng) FromOptions(options []option) error {
	if len(options) != 0 {
		return fmt.Errorf("Unknown options for virtio-rng devices: %s", options)
	}
	return nil
}

// VirtioBlkNew creates a new disk to use in the virtual machine. It will use
// the file at imagePath as the disk image. This image must be in raw format.
func VirtioBlkNew(imagePath string) (VirtioDevice, error) {
	return &VirtioBlk{
		ImagePath: imagePath,
	}, nil
}

func (dev *VirtioBlk) ToCmdLine() ([]string, error) {
	if dev.ImagePath == "" {
		return nil, fmt.Errorf("virtio-blk needs the path to a disk image")
	}
	return []string{"--device", fmt.Sprintf("virtio-blk,path=%s", dev.ImagePath)}, nil
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

// VirtioVsockNew creates a new virtio-vsock device for 2-way communication
// between the host and the virtual machine. The communication will happen on
// vsock port, and on the host it will use the unix socket at socketURL.
// When listen is true, the host will be listening for connections over vsock.
// When listen  is false, the guest will be listening for connections over vsock.
func VirtioVsockNew(port uint, socketURL string, listen bool) (VirtioDevice, error) {
	return &VirtioVsock{
		Port:      port,
		SocketURL: socketURL,
		Listen:    listen,
	}, nil
}

func (dev *VirtioVsock) ToCmdLine() ([]string, error) {
	if dev.Port == 0 || dev.SocketURL == "" {
		return nil, fmt.Errorf("virtio-vsock needs both a port and a socket URL")
	}
	var listenStr string
	if dev.Listen {
		listenStr = "listen"
	} else {
		listenStr = "connect"
	}
	return []string{"--device", fmt.Sprintf("virtio-vsock,port=%d,socketURL=%s,%s", dev.Port, dev.SocketURL, listenStr)}, nil
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

// VirtioFsNew creates a new virtio-fs device for file sharing. It will share
// the directory at sharedDir with the virtual machine. This directory can be
// mounted in the VM using `mount -t virtiofs mountTag /some/dir`
func VirtioFsNew(sharedDir string, mountTag string) (VirtioDevice, error) {
	return &VirtioFs{
		SharedDir: sharedDir,
		MountTag:  mountTag,
	}, nil
}

func (dev *VirtioFs) ToCmdLine() ([]string, error) {
	if dev.SharedDir == "" {
		return nil, fmt.Errorf("virtio-fs needs the path to the directory to share")
	}
	if dev.MountTag != "" {
		return []string{"--device", fmt.Sprintf("virtio-fs,sharedDir=%s,mountTag=%s", dev.SharedDir, dev.MountTag)}, nil
	} else {
		return []string{"--device", fmt.Sprintf("virtio-fs,sharedDir=%s", dev.SharedDir)}, nil
	}
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
