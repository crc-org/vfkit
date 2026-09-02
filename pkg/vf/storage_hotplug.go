package vf

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrStorageDeviceConflict is returned when a device ID is already used by
	// a device with a different configuration.
	ErrStorageDeviceConflict = errors.New("storage device ID is already in use")
	// ErrStorageDeviceBusy is returned when an attach or detach for the same
	// device ID is still in progress.
	ErrStorageDeviceBusy = errors.New("storage device operation is already in progress")
	// ErrInvalidStorageDevice wraps errors caused by an unusable device
	// specification (malformed NBD URI, unsuitable raw disk image, ...).
	ErrInvalidStorageDevice = errors.New("invalid storage device")
	// ErrUSBControllerUnavailable is returned when the virtual machine has no
	// usb-xhci controller to attach devices to.
	ErrUSBControllerUnavailable = errors.New("usb-xhci controller is not configured")

	errStorageDeviceNotAttached = errors.New("storage device is not attached")
)

// States reported by HotpluggedStorageDevice.State.
const (
	// StorageDeviceStateAttached is the state of a raw-backed device.
	StorageDeviceStateAttached = "attached"
	// StorageDeviceStateConnecting is the state of an NBD-backed device until
	// the NBD client reports a connection.
	StorageDeviceStateConnecting = "connecting"
	// StorageDeviceStateConnected means the NBD client is connected.
	StorageDeviceStateConnected = "connected"
	// StorageDeviceStateDisconnected means the NBD client reported an error
	// and is trying to reconnect.
	StorageDeviceStateDisconnected = "disconnected"
)

type NBDStorageDevice struct {
	URI                 string
	Timeout             time.Duration
	SynchronizationMode config.NBDSynchronizationMode
	ReadOnly            bool
}

type RawStorageDevice struct {
	Path     string
	ReadOnly bool
}

type HotpluggedStorageDevice struct {
	ID       string `json:"id"`
	Backend  string `json:"backend"`
	ReadOnly bool   `json:"readOnly"`
	State    string `json:"state"`
}

// RedactLocation replaces location (an NBD URI or a raw disk path) in detail
// with a placeholder so the text can be logged or returned to clients.
func RedactLocation(detail, location string) string {
	if location == "" {
		return detail
	}
	return strings.ReplaceAll(detail, location, "<redacted>")
}

type storageHotplugHandle interface {
	Detach() error
	Connected() <-chan struct{}
	DidEncounterError() <-chan error
}

type storageHotplugBackend interface {
	AttachNBDStorageDevice(spec NBDStorageDevice) (storageHotplugHandle, error)
	AttachRawStorageDevice(spec RawStorageDevice) (storageHotplugHandle, error)
}

type storageDeviceSpec struct {
	backend             string
	location            string
	timeout             time.Duration
	synchronizationMode config.NBDSynchronizationMode
	readOnly            bool
}

type hotpluggedStorageDevice struct {
	spec   storageDeviceSpec
	handle storageHotplugHandle
	state  string
	// pending is set while an attach or detach is in flight. The manager lock
	// is released during those calls, which block until the virtualization
	// framework has finished.
	pending     bool
	stopMonitor chan struct{}
}

type storageHotplugManager struct {
	mu      sync.Mutex
	backend storageHotplugBackend
	devices map[string]*hotpluggedStorageDevice
}

func newStorageHotplugManager(backend storageHotplugBackend) *storageHotplugManager {
	return &storageHotplugManager{
		backend: backend,
		devices: make(map[string]*hotpluggedStorageDevice),
	}
}

func (vm *VirtualMachine) HotpluggedStorageDevices() []HotpluggedStorageDevice {
	if vm.storageHotplug == nil {
		return []HotpluggedStorageDevice{}
	}
	return vm.storageHotplug.List()
}

func (vm *VirtualMachine) HotplugNBDStorageDevice(id string, spec NBDStorageDevice) (bool, error) {
	if vm.storageHotplug == nil {
		return false, ErrUSBControllerUnavailable
	}
	return vm.storageHotplug.Attach(id, spec)
}

func (vm *VirtualMachine) HotplugRawStorageDevice(id string, spec RawStorageDevice) (bool, error) {
	if vm.storageHotplug == nil {
		return false, ErrUSBControllerUnavailable
	}
	return vm.storageHotplug.AttachRaw(id, spec)
}

func (vm *VirtualMachine) DetachHotpluggedStorageDevice(id string) (bool, error) {
	if vm.storageHotplug == nil {
		return false, nil
	}
	return vm.storageHotplug.Detach(id)
}

func (manager *storageHotplugManager) List() []HotpluggedStorageDevice {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	devices := make([]HotpluggedStorageDevice, 0, len(manager.devices))
	for id, device := range manager.devices {
		if device.handle == nil {
			// attach still in progress
			continue
		}
		devices = append(devices, HotpluggedStorageDevice{
			ID:       id,
			Backend:  device.spec.backend,
			ReadOnly: device.spec.readOnly,
			State:    device.state,
		})
	}
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})
	return devices
}

func (manager *storageHotplugManager) Attach(id string, spec NBDStorageDevice) (bool, error) {
	deviceSpec := storageDeviceSpec{
		backend:             "nbd",
		location:            spec.URI,
		timeout:             spec.Timeout,
		synchronizationMode: spec.SynchronizationMode,
		readOnly:            spec.ReadOnly,
	}
	return manager.attach(id, deviceSpec, func() (storageHotplugHandle, error) {
		return manager.backend.AttachNBDStorageDevice(spec)
	}, true)
}

func (manager *storageHotplugManager) AttachRaw(id string, spec RawStorageDevice) (bool, error) {
	deviceSpec := storageDeviceSpec{
		backend:  "raw",
		location: spec.Path,
		readOnly: spec.ReadOnly,
	}
	return manager.attach(id, deviceSpec, func() (storageHotplugHandle, error) {
		return manager.backend.AttachRawStorageDevice(spec)
	}, false)
}

// attach reserves id, runs the backend attach without holding the manager lock
// and then records the result, so that listing and operations on other
// devices are not blocked behind a slow attach.
func (manager *storageHotplugManager) attach(
	id string,
	spec storageDeviceSpec,
	attach func() (storageHotplugHandle, error),
	monitor bool,
) (bool, error) {
	record, err := manager.reserve(id, spec)
	if err != nil || record == nil {
		return false, err
	}

	handle, err := attach()

	manager.mu.Lock()
	defer manager.mu.Unlock()
	if err != nil {
		delete(manager.devices, id)
		return false, err
	}
	record.handle = handle
	record.pending = false
	record.state = StorageDeviceStateAttached
	if monitor {
		record.state = StorageDeviceStateConnecting
		record.stopMonitor = make(chan struct{})
		go manager.monitorNBDStorageDevice(id, record, record.stopMonitor)
	}
	log.Infof("Attached %s-backed USB storage device %s", spec.backend, id)
	return true, nil
}

// reserve claims id for an attach in progress. It returns a nil record and a
// nil error when an identical device is already attached.
func (manager *storageHotplugManager) reserve(id string, spec storageDeviceSpec) (*hotpluggedStorageDevice, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if current, ok := manager.devices[id]; ok {
		if current.pending {
			return nil, ErrStorageDeviceBusy
		}
		if current.spec == spec {
			return nil, nil
		}
		return nil, ErrStorageDeviceConflict
	}
	record := &hotpluggedStorageDevice{spec: spec, pending: true}
	manager.devices[id] = record
	return record, nil
}

func (manager *storageHotplugManager) Detach(id string) (bool, error) {
	manager.mu.Lock()
	record, ok := manager.devices[id]
	if !ok {
		manager.mu.Unlock()
		return false, nil
	}
	if record.pending {
		manager.mu.Unlock()
		return false, ErrStorageDeviceBusy
	}
	record.pending = true
	manager.mu.Unlock()

	err := record.handle.Detach()

	manager.mu.Lock()
	defer manager.mu.Unlock()
	if errors.Is(err, errStorageDeviceNotAttached) {
		log.Warnf("USB storage device %s is no longer attached, dropping it", id)
		err = nil
	}
	if err != nil {
		record.pending = false
		return false, err
	}
	delete(manager.devices, id)
	if record.stopMonitor != nil {
		close(record.stopMonitor)
	}
	log.Infof("Detached %s-backed USB storage device %s", record.spec.backend, id)
	return true, nil
}

func (manager *storageHotplugManager) setState(record *hotpluggedStorageDevice, state string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	record.state = state
}

func (manager *storageHotplugManager) monitorNBDStorageDevice(
	id string,
	record *hotpluggedStorageDevice,
	stop <-chan struct{},
) {
	// handle and spec are set before the goroutine is started and never change.
	handle := record.handle
	uri := record.spec.location
	for {
		select {
		case err, ok := <-handle.DidEncounterError():
			if !ok {
				return
			}
			detail := "unknown error"
			if err != nil {
				detail = RedactLocation(err.Error(), uri)
			}
			manager.setState(record, StorageDeviceStateDisconnected)
			log.Warnf("NBD-backed USB storage device %s disconnected: %s", id, detail)
		case _, ok := <-handle.Connected():
			if !ok {
				return
			}
			manager.setState(record, StorageDeviceStateConnected)
			log.Infof("NBD-backed USB storage device %s connected", id)
		case <-stop:
			return
		}
	}
}

type vzStorageHotplugBackend struct {
	vm *vz.VirtualMachine
}

func (backend *vzStorageHotplugBackend) usbController() (*vz.USBController, error) {
	controllers := backend.vm.USBControllers()
	if len(controllers) != 1 {
		return nil, ErrUSBControllerUnavailable
	}
	return controllers[0], nil
}

func (backend *vzStorageHotplugBackend) AttachNBDStorageDevice(
	spec NBDStorageDevice,
) (storageHotplugHandle, error) {
	controller, err := backend.usbController()
	if err != nil {
		return nil, err
	}
	if err := config.ValidateNBDURI(spec.URI); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidStorageDevice, err)
	}
	attachment, err := vz.NewNetworkBlockDeviceStorageDeviceAttachment(
		spec.URI,
		spec.Timeout,
		spec.ReadOnly,
		nbdSynchronizationModeVZ(spec.SynchronizationMode),
	)
	if err != nil {
		return nil, wrapAttachmentError("could not create NBD attachment", err)
	}
	handle, err := attachUSBStorageDevice(controller, attachment)
	if err != nil {
		return nil, err
	}
	handle.connected = attachment.Connected()
	handle.didEncounterError = attachment.DidEncounterError()
	return handle, nil
}

func (backend *vzStorageHotplugBackend) AttachRawStorageDevice(
	spec RawStorageDevice,
) (storageHotplugHandle, error) {
	controller, err := backend.usbController()
	if err != nil {
		return nil, err
	}
	if err := validateRawStorageDevice(spec.Path); err != nil {
		return nil, err
	}
	storageConfig := DiskStorageConfig{
		StorageConfig: config.StorageConfig{DevName: "usb-mass-storage", ReadOnly: spec.ReadOnly},
		ImagePath:     spec.Path,
	}
	attachment, err := storageConfig.toVz()
	if err != nil {
		return nil, wrapAttachmentError("could not create raw disk attachment", err)
	}
	handle, err := attachUSBStorageDevice(controller, attachment)
	if err != nil {
		return nil, err
	}
	return handle, nil
}

func validateRawStorageDevice(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%w: could not inspect raw disk image: %w", ErrInvalidStorageDevice, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: raw disk image is not a regular file", ErrInvalidStorageDevice)
	}
	if info.Size() == 0 || info.Size()%512 != 0 {
		return fmt.Errorf("%w: raw disk image size must be a positive multiple of 512 bytes", ErrInvalidStorageDevice)
	}
	return nil
}

// wrapAttachmentError marks framework errors caused by an unusable device
// specification as ErrInvalidStorageDevice so callers can report them as
// client errors rather than internal failures.
func wrapAttachmentError(message string, err error) error {
	if isVZError(err, vz.ErrorInvalidVirtualMachineConfiguration) || isVZError(err, vz.ErrorInvalidDiskImage) {
		return fmt.Errorf("%w: %s: %w", ErrInvalidStorageDevice, message, err)
	}
	return fmt.Errorf("%s: %w", message, err)
}

func isVZError(err error, code vz.ErrorCode) bool {
	var nsErr *vz.NSError
	return errors.As(err, &nsErr) && nsErr.Domain == "VZErrorDomain" && nsErr.Code == int(code)
}

// attachUSBStorageDevice wraps attachment in a USB mass-storage device and
// attaches it to controller.
//
// vz v3.7.1 never releases VZUSBMassStorageDevice objects and the framework
// refuses to re-attach a detached device, so every attach/detach cycle retains
// one device object (with the configuration and attachment it references)
// until vfkit exits. The controller lookup and specification validation run
// before any framework object is created to keep that set to devices the
// framework actually attached.
func attachUSBStorageDevice(
	controller *vz.USBController,
	attachment vz.StorageDeviceAttachment,
) (*vzStorageHotplugHandle, error) {
	configuration, err := vz.NewUSBMassStorageDeviceConfiguration(attachment)
	if err != nil {
		return nil, fmt.Errorf("could not create USB mass-storage configuration: %w", err)
	}
	device, err := vz.NewUSBMassStorageDevice(configuration)
	if err != nil {
		return nil, fmt.Errorf("could not create USB mass-storage device: %w", err)
	}
	handle := &vzStorageHotplugHandle{
		controller:    controller,
		device:        device,
		configuration: configuration,
		attachment:    attachment,
	}
	if err := controller.Attach(device); err != nil {
		return nil, fmt.Errorf("could not attach USB mass-storage device: %w", err)
	}
	return handle, nil
}

type vzStorageHotplugHandle struct {
	controller *vz.USBController
	device     vz.USBDevice
	// configuration and attachment are never read: they keep the Go wrappers,
	// and through their finalizers the Objective-C objects, alive for as long
	// as the device is attached.
	configuration     *vz.USBMassStorageDeviceConfiguration
	attachment        vz.StorageDeviceAttachment
	connected         <-chan struct{}
	didEncounterError <-chan error
}

func (handle *vzStorageHotplugHandle) Detach() error {
	err := handle.controller.Detach(handle.device)
	if err == nil {
		return nil
	}
	if isVZError(err, vz.ErrorDeviceNotFound) {
		return fmt.Errorf("%w: %w", errStorageDeviceNotAttached, err)
	}
	return fmt.Errorf("could not detach USB mass-storage device: %w", err)
}

func (handle *vzStorageHotplugHandle) Connected() <-chan struct{} {
	return handle.connected
}

func (handle *vzStorageHotplugHandle) DidEncounterError() <-chan error {
	return handle.didEncounterError
}
