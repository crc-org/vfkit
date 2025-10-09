package vf

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/crc-org/vfkit/pkg/config"
	"github.com/crc-org/vfkit/pkg/util"
	"golang.org/x/sys/unix"

	"github.com/Code-Hex/vz/v3"
	"github.com/pkg/term/termios"
	log "github.com/sirupsen/logrus"
)

// vf will define toVZ() and AddToVirtualMachineConfig() methods on these types
// We alias the types from the config package to avoid duplicating struct
// definitions between the config and vf packages
type RosettaShare config.RosettaShare
type NVMExpressController config.NVMExpressController
type VirtioBlk config.VirtioBlk
type VirtioFs config.VirtioFs
type VirtioRng config.VirtioRng
type VirtioSerial config.VirtioSerial
type VirtioVsock config.VirtioVsock
type VirtioInput config.VirtioInput
type VirtioGPU config.VirtioGPU
type VirtioBalloon config.VirtioBalloon
type NetworkBlockDevice config.NetworkBlockDevice

type vzNetworkBlockDevice struct {
	*vz.VirtioBlockDeviceConfiguration
	config *NetworkBlockDevice
}

func (dev *NVMExpressController) toVz() (vz.StorageDeviceConfiguration, error) {
	var storageConfig = DiskStorageConfig(dev.DiskStorageConfig)
	attachment, err := storageConfig.toVz()
	if err != nil {
		return nil, err
	}
	devConfig, err := vz.NewNVMExpressControllerDeviceConfiguration(attachment)
	if err != nil {
		return nil, err
	}

	return devConfig, nil
}

func (dev *NVMExpressController) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	storageDeviceConfig, err := dev.toVz()
	if err != nil {
		return err
	}
	log.Infof("Adding nvme device (imagePath: %s)", dev.ImagePath)
	vmConfig.storageDevicesConfiguration = append(vmConfig.storageDevicesConfiguration, storageDeviceConfig)

	return nil
}

func (dev *VirtioBlk) toVz() (vz.StorageDeviceConfiguration, error) {
	var storageConfig = DiskStorageConfig(dev.DiskStorageConfig)
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

func (dev *VirtioBlk) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	storageDeviceConfig, err := dev.toVz()
	if err != nil {
		return err
	}
	log.Infof("Adding virtio-blk device (imagePath: %s)", dev.ImagePath)
	vmConfig.storageDevicesConfiguration = append(vmConfig.storageDevicesConfiguration, storageDeviceConfig)

	return nil
}

func (dev *VirtioInput) toVz() (interface{}, error) {
	var inputConfig interface{}
	if dev.InputType == config.VirtioInputPointingDevice {
		inputConfig, err := vz.NewUSBScreenCoordinatePointingDeviceConfiguration()
		if err != nil {
			return nil, fmt.Errorf("failed to create pointing device configuration: %w", err)
		}

		return inputConfig, nil
	}

	inputConfig, err := vz.NewUSBKeyboardConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to create keyboard device configuration: %w", err)
	}

	return inputConfig, nil
}

func (dev *VirtioInput) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	inputDeviceConfig, err := dev.toVz()
	if err != nil {
		return err
	}

	switch conf := inputDeviceConfig.(type) {
	case vz.PointingDeviceConfiguration:
		log.Info("Adding virtio-input pointing device")
		vmConfig.pointingDevicesConfiguration = append(vmConfig.pointingDevicesConfiguration, conf)
	case vz.KeyboardConfiguration:
		log.Info("Adding virtio-input keyboard device")
		vmConfig.keyboardConfiguration = append(vmConfig.keyboardConfiguration, conf)
	}

	return nil
}

func newVirtioGraphicsDeviceConfiguration(dev *VirtioGPU) (vz.GraphicsDeviceConfiguration, error) {
	gpuDeviceConfig, err := vz.NewVirtioGraphicsDeviceConfiguration()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize virtio graphics device: %w", err)
	}
	graphicsScanoutConfig, err := vz.NewVirtioGraphicsScanoutConfiguration(int64(dev.Width), int64(dev.Height))

	if err != nil {
		return nil, fmt.Errorf("failed to create graphics scanout: %w", err)
	}

	gpuDeviceConfig.SetScanouts(
		graphicsScanoutConfig,
	)

	return gpuDeviceConfig, nil
}

func (dev *VirtioGPU) toVz() (vz.GraphicsDeviceConfiguration, error) {
	log.Debugf("Setting up graphics device with %vx%v resolution.", dev.Width, dev.Height)

	if PlatformType == "macos" {
		return newMacGraphicsDeviceConfiguration(dev)
	}
	return newVirtioGraphicsDeviceConfiguration(dev)

}

func (dev *VirtioGPU) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	gpuDeviceConfig, err := dev.toVz()
	if err != nil {
		return err
	}

	log.Infof("Adding virtio-gpu device")

	vmConfig.graphicsDevicesConfiguration = append(vmConfig.graphicsDevicesConfiguration, gpuDeviceConfig)

	return nil
}

func (dev *VirtioFs) toVz() (vz.DirectorySharingDeviceConfiguration, error) {
	if dev.SharedDir == "" {
		return nil, fmt.Errorf("missing mandatory 'sharedDir' option for virtio-fs device")
	}
	var mountTag string
	if dev.MountTag != "" {
		mountTag = dev.MountTag
	} else {
		mountTag = filepath.Base(dev.SharedDir)
	}

	sharedDir, err := vz.NewSharedDirectory(dev.SharedDir, false)
	if err != nil {
		return nil, err
	}
	sharedDirConfig, err := vz.NewSingleDirectoryShare(sharedDir)
	if err != nil {
		return nil, err
	}
	fileSystemDeviceConfig, err := vz.NewVirtioFileSystemDeviceConfiguration(mountTag)
	if err != nil {
		return nil, err
	}
	fileSystemDeviceConfig.SetDirectoryShare(sharedDirConfig)

	return fileSystemDeviceConfig, nil
}

func (dev *VirtioFs) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	fileSystemDeviceConfig, err := dev.toVz()
	if err != nil {
		return err
	}
	log.Infof("Adding virtio-fs device")
	vmConfig.directorySharingDevicesConfiguration = append(vmConfig.directorySharingDevicesConfiguration, fileSystemDeviceConfig)
	return nil
}

func (dev *VirtioRng) toVz() (*vz.VirtioEntropyDeviceConfiguration, error) {
	return vz.NewVirtioEntropyDeviceConfiguration()
}

func (dev *VirtioRng) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	log.Infof("Adding virtio-rng device")
	entropyConfig, err := dev.toVz()
	if err != nil {
		return err
	}
	vmConfig.entropyDevicesConfiguration = append(vmConfig.entropyDevicesConfiguration, entropyConfig)

	return nil
}

func (dev *VirtioBalloon) toVz() (*vz.VirtioTraditionalMemoryBalloonDeviceConfiguration, error) {
	return vz.NewVirtioTraditionalMemoryBalloonDeviceConfiguration()
}

func (dev *VirtioBalloon) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	log.Infof("Adding virtio-balloon device")
	balloonConfig, err := dev.toVz()
	if err != nil {
		return err
	}
	vmConfig.SetMemoryBalloonDevicesVirtualMachineConfiguration([]vz.MemoryBalloonDeviceConfiguration{balloonConfig})

	return nil
}

func unixFd(fd uintptr) int {
	// On unix the underlying fd is int, overflow is not possible.
	return int(fd) //#nosec G115 -- potential integer overflow
}

// https://developer.apple.com/documentation/virtualization/running_linux_in_a_virtual_machine#3880009
func setRawMode(f *os.File) error {
	// Get settings for terminal
	var attr unix.Termios
	if err := termios.Tcgetattr(f.Fd(), &attr); err != nil {
		return err
	}

	// Put stdin into raw mode, disabling local echo, input canonicalization,
	// and CR-NL mapping.
	attr.Iflag &^= unix.ICRNL
	attr.Lflag &^= unix.ICANON | unix.ECHO

	// reflects the changed settings
	return termios.Tcsetattr(f.Fd(), termios.TCSANOW, &attr)
}

func (dev *VirtioSerial) toVz() (*vz.VirtioConsoleDeviceSerialPortConfiguration, error) {
	var serialPortAttachment vz.SerialPortAttachment
	var retErr error
	switch {
	case dev.UsesStdio:
		if err := setRawMode(os.Stdin); err != nil {
			return nil, err
		}
		serialPortAttachment, retErr = vz.NewFileHandleSerialPortAttachment(os.Stdin, os.Stdout)
	default:
		serialPortAttachment, retErr = vz.NewFileSerialPortAttachment(dev.LogFile, false)
	}
	if retErr != nil {
		return nil, retErr
	}

	return vz.NewVirtioConsoleDeviceSerialPortConfiguration(serialPortAttachment)
}

func (dev *VirtioSerial) toVzConsole() (*vz.VirtioConsolePortConfiguration, error) {
	master, slave, err := termios.Pty()
	if err != nil {
		return nil, err
	}

	// the master fd and slave fd must stay open for vfkit's lifetime
	util.RegisterExitHandler(func() {
		_ = master.Close()
		_ = slave.Close()
	})

	dev.PtyName = slave.Name()

	if err := setRawMode(master); err != nil {
		return nil, err
	}
	serialPortAttachment, retErr := vz.NewFileHandleSerialPortAttachment(master, master)
	if retErr != nil {
		return nil, retErr
	}
	return vz.NewVirtioConsolePortConfiguration(
		vz.WithVirtioConsolePortConfigurationAttachment(serialPortAttachment),
		vz.WithVirtioConsolePortConfigurationIsConsole(true))
}

func (dev *VirtioSerial) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	if dev.LogFile != "" {
		log.Infof("Adding virtio-serial device (logFile: %s)", dev.LogFile)
	}
	if dev.UsesStdio {
		log.Infof("Adding stdio console")
	}
	if dev.PtyName != "" {
		return fmt.Errorf("VirtioSerial.PtyName must be empty (current value: %s)", dev.PtyName)
	}

	if dev.UsesPty {
		consolePortConfig, err := dev.toVzConsole()
		if err != nil {
			return err
		}
		vmConfig.consolePortsConfiguration = append(vmConfig.consolePortsConfiguration, consolePortConfig)
		log.Infof("Using PTY (pty path: %s)", dev.PtyName)
	} else {
		consoleConfig, err := dev.toVz()
		if err != nil {
			return err
		}
		vmConfig.serialPortsConfiguration = append(vmConfig.serialPortsConfiguration, consoleConfig)
	}

	return nil
}

func (dev *VirtioVsock) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	if len(vmConfig.socketDevicesConfiguration) != 0 {
		log.Debugf("virtio-vsock device already present, not adding a second one")
		return nil
	}
	log.Infof("Adding virtio-vsock device")
	vzdev, err := vz.NewVirtioSocketDeviceConfiguration()
	if err != nil {
		return err
	}
	vmConfig.socketDevicesConfiguration = append(vmConfig.socketDevicesConfiguration, vzdev)

	return nil
}

func (dev *NetworkBlockDevice) toVz() (vz.StorageDeviceConfiguration, error) {
	if err := dev.validateNbdURI(dev.URI); err != nil {
		return nil, fmt.Errorf("invalid NBD device 'uri': %s", err.Error())
	}

	if err := dev.validateNbdDeviceIdentifier(dev.DeviceIdentifier); err != nil {
		return nil, fmt.Errorf("invalid NBD device 'deviceId': %s", err.Error())
	}

	attachment, err := vz.NewNetworkBlockDeviceStorageDeviceAttachment(dev.URI, dev.Timeout, dev.ReadOnly, dev.SynchronizationModeVZ())
	if err != nil {
		return nil, err
	}

	vzdev, err := vz.NewVirtioBlockDeviceConfiguration(attachment)
	if err != nil {
		return nil, err
	}
	err = vzdev.SetBlockDeviceIdentifier(dev.DeviceIdentifier)
	if err != nil {
		return nil, err
	}

	return vzNetworkBlockDevice{VirtioBlockDeviceConfiguration: vzdev, config: dev}, nil
}

func (dev *NetworkBlockDevice) validateNbdURI(uri string) error {
	if uri == "" {
		return fmt.Errorf("'uri' must be specified")
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	// The format specified by https://github.com/NetworkBlockDevice/nbd/blob/master/doc/uri.md
	if parsed.Scheme != "nbd" && parsed.Scheme != "nbds" && parsed.Scheme != "nbd+unix" && parsed.Scheme != "nbds+unix" {
		return fmt.Errorf("invalid scheme: %s. Expected one of: 'nbd', 'nbds', 'nbd+unix', or 'nbds+unix'", parsed.Scheme)
	}

	return nil
}

func (dev *NetworkBlockDevice) validateNbdDeviceIdentifier(deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("'deviceId' must be specified")
	}

	if strings.Contains(deviceID, "/") {
		return fmt.Errorf("invalid 'deviceId': it cannot contain any forward slash")
	}

	if len(deviceID) > 255 {
		return fmt.Errorf("invalid 'deviceId': exceeds maximum length")
	}

	return nil
}

func (dev *NetworkBlockDevice) SynchronizationModeVZ() vz.DiskSynchronizationMode {
	if dev.SynchronizationMode == config.SynchronizationNoneMode {
		return vz.DiskSynchronizationModeNone
	}
	return vz.DiskSynchronizationModeFull
}

func (dev *NetworkBlockDevice) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	storageDeviceConfig, err := dev.toVz()
	if err != nil {
		return err
	}
	log.Infof("Adding NBD device (uri: %s, deviceId: %s)", dev.URI, dev.DeviceIdentifier)
	vmConfig.storageDevicesConfiguration = append(vmConfig.storageDevicesConfiguration, storageDeviceConfig)

	return nil
}

func ListenNetworkBlockDevices(vm *VirtualMachine) error {
	for _, dev := range vm.vfConfig.storageDevicesConfiguration {
		if nbdDev, isNbdDev := dev.(vzNetworkBlockDevice); isNbdDev {
			nbdAttachment, isNbdAttachment := dev.Attachment().(*vz.NetworkBlockDeviceStorageDeviceAttachment)
			if !isNbdAttachment {
				log.Info("Found NBD device with no NBD attachment. Please file a vfkit bug.")
				return fmt.Errorf("NetworkBlockDevice must use a NBD attachment")
			}
			nbdConfig := nbdDev.config
			go func() {
				for {
					select {
					case err := <-nbdAttachment.DidEncounterError():
						log.Infof("Disconnected from NBD server %s. Error %v", nbdConfig.URI, err.Error())
					case <-nbdAttachment.Connected():
						log.Infof("Successfully connected to NBD server %s.", nbdConfig.URI)
					}
				}
			}()
		}
	}
	return nil
}

func AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration, dev config.VirtioDevice) error {
	switch d := dev.(type) {
	case *config.USBMassStorage:
		return (*USBMassStorage)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioBlk:
		return (*VirtioBlk)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.RosettaShare:
		return (*RosettaShare)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.NVMExpressController:
		return (*NVMExpressController)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioFs:
		return (*VirtioFs)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioNet:
		dev := VirtioNet{VirtioNet: d}
		return dev.AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioRng:
		return (*VirtioRng)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioSerial:
		return (*VirtioSerial)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioVsock:
		return (*VirtioVsock)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioInput:
		return (*VirtioInput)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioGPU:
		return (*VirtioGPU)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.VirtioBalloon:
		return (*VirtioBalloon)(d).AddToVirtualMachineConfig(vmConfig)
	case *config.NetworkBlockDevice:
		return (*NetworkBlockDevice)(d).AddToVirtualMachineConfig(vmConfig)
	default:
		return fmt.Errorf("unexpected virtio device type: %T", d)
	}
}

func (config *DiskStorageConfig) toVz() (vz.StorageDeviceAttachment, error) {
	if config.ImagePath == "" {
		return nil, fmt.Errorf("missing mandatory 'path' option for %s device", config.DevName)
	}
	return vz.NewDiskImageStorageDeviceAttachment(config.ImagePath, config.ReadOnly)
}

func (dev *USBMassStorage) toVz() (vz.StorageDeviceConfiguration, error) {
	var storageConfig = DiskStorageConfig(dev.DiskStorageConfig)
	attachment, err := storageConfig.toVz()
	if err != nil {
		return nil, err
	}
	return vz.NewUSBMassStorageDeviceConfiguration(attachment)
}

func (dev *USBMassStorage) AddToVirtualMachineConfig(vmConfig *VirtualMachineConfiguration) error {
	storageDeviceConfig, err := dev.toVz()
	if err != nil {
		return err
	}
	log.Infof("Adding USB mass storage device (imagePath: %s)", dev.ImagePath)
	vmConfig.storageDevicesConfiguration = append(vmConfig.storageDevicesConfiguration, storageDeviceConfig)

	return nil
}

type DiskStorageConfig config.DiskStorageConfig

type USBMassStorage config.USBMassStorage
