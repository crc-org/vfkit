package vf

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/crc-org/vfkit/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStorageHotplugHandle struct {
	connected chan struct{}
	errCh     chan error
	detachErr error
	detaches  int
}

func newFakeStorageHotplugHandle() *fakeStorageHotplugHandle {
	return &fakeStorageHotplugHandle{
		connected: make(chan struct{}),
		errCh:     make(chan error),
	}
}

func (handle *fakeStorageHotplugHandle) Detach() error {
	handle.detaches++
	return handle.detachErr
}

func (handle *fakeStorageHotplugHandle) Connected() <-chan struct{} {
	return handle.connected
}

func (handle *fakeStorageHotplugHandle) DidEncounterError() <-chan error {
	return handle.errCh
}

type fakeStorageHotplugBackend struct {
	handles     []*fakeStorageHotplugHandle
	attachErr   error
	attachCalls []NBDStorageDevice
	rawCalls    []RawStorageDevice
	// attachStarted, when set, receives a value when an attach begins;
	// attachBlock, when set, must be closed before the attach completes.
	attachStarted chan struct{}
	attachBlock   chan struct{}
}

func (backend *fakeStorageHotplugBackend) attach() (storageHotplugHandle, error) {
	if backend.attachStarted != nil {
		backend.attachStarted <- struct{}{}
	}
	if backend.attachBlock != nil {
		<-backend.attachBlock
	}
	if backend.attachErr != nil {
		return nil, backend.attachErr
	}
	handle := newFakeStorageHotplugHandle()
	backend.handles = append(backend.handles, handle)
	return handle, nil
}

func (backend *fakeStorageHotplugBackend) AttachNBDStorageDevice(
	spec NBDStorageDevice,
) (storageHotplugHandle, error) {
	backend.attachCalls = append(backend.attachCalls, spec)
	return backend.attach()
}

func (backend *fakeStorageHotplugBackend) AttachRawStorageDevice(
	spec RawStorageDevice,
) (storageHotplugHandle, error) {
	backend.rawCalls = append(backend.rawCalls, spec)
	return backend.attach()
}

func testNBDStorageDevice(uri string) NBDStorageDevice {
	return NBDStorageDevice{
		URI:                 uri,
		Timeout:             60 * time.Second,
		SynchronizationMode: config.SynchronizationFullMode,
	}
}

func TestStorageHotplugManagerAttachIsIdempotent(t *testing.T) {
	backend := &fakeStorageHotplugBackend{}
	manager := newStorageHotplugManager(backend)
	spec := testNBDStorageDevice("nbd://192.0.2.10:10809/export")

	created, err := manager.Attach("cache", spec)
	require.NoError(t, err)
	assert.True(t, created)
	created, err = manager.Attach("cache", spec)
	require.NoError(t, err)
	assert.False(t, created)
	assert.Len(t, backend.attachCalls, 1)

	_, err = manager.Detach("cache")
	require.NoError(t, err)
}

func TestStorageHotplugManagerRejectsConflictingID(t *testing.T) {
	backend := &fakeStorageHotplugBackend{}
	manager := newStorageHotplugManager(backend)
	_, err := manager.Attach("cache", testNBDStorageDevice("nbd://192.0.2.10:10809/first"))
	require.NoError(t, err)

	created, err := manager.Attach("cache", testNBDStorageDevice("nbd://192.0.2.10:10809/second"))
	assert.False(t, created)
	assert.ErrorIs(t, err, ErrStorageDeviceConflict)
	assert.Len(t, backend.attachCalls, 1)

	_, err = manager.Detach("cache")
	require.NoError(t, err)
}

func TestStorageHotplugManagerForgetsFailedAttach(t *testing.T) {
	backend := &fakeStorageHotplugBackend{attachErr: errors.New("attach failed")}
	manager := newStorageHotplugManager(backend)

	created, err := manager.Attach("cache", testNBDStorageDevice("nbd://192.0.2.10:10809/export"))
	assert.False(t, created)
	assert.EqualError(t, err, "attach failed")
	assert.Empty(t, manager.List())

	backend.attachErr = nil
	created, err = manager.Attach("cache", testNBDStorageDevice("nbd://192.0.2.10:10809/export"))
	require.NoError(t, err)
	assert.True(t, created)
	_, err = manager.Detach("cache")
	require.NoError(t, err)
}

func TestStorageHotplugManagerRetainsDeviceAfterDetachFailure(t *testing.T) {
	backend := &fakeStorageHotplugBackend{}
	manager := newStorageHotplugManager(backend)
	_, err := manager.Attach("cache", testNBDStorageDevice("nbd://192.0.2.10:10809/export"))
	require.NoError(t, err)
	backend.handles[0].detachErr = errors.New("detach failed")

	detached, err := manager.Detach("cache")
	assert.False(t, detached)
	assert.EqualError(t, err, "detach failed")
	assert.Equal(t, []HotpluggedStorageDevice{
		{ID: "cache", Backend: "nbd", State: StorageDeviceStateConnecting},
	}, manager.List())

	backend.handles[0].detachErr = nil
	_, err = manager.Detach("cache")
	require.NoError(t, err)
	assert.Empty(t, manager.List())
}

func TestStorageHotplugManagerDropsDeviceThatIsNoLongerAttached(t *testing.T) {
	backend := &fakeStorageHotplugBackend{}
	manager := newStorageHotplugManager(backend)
	_, err := manager.Attach("cache", testNBDStorageDevice("nbd://192.0.2.10:10809/export"))
	require.NoError(t, err)
	backend.handles[0].detachErr = fmt.Errorf("device gone: %w", errStorageDeviceNotAttached)

	detached, err := manager.Detach("cache")
	require.NoError(t, err)
	assert.True(t, detached)
	assert.Empty(t, manager.List())
}

func TestStorageHotplugManagerDoesNotBlockWhileAttaching(t *testing.T) {
	backend := &fakeStorageHotplugBackend{
		attachStarted: make(chan struct{}),
		attachBlock:   make(chan struct{}),
	}
	manager := newStorageHotplugManager(backend)
	spec := testNBDStorageDevice("nbd://192.0.2.10:10809/export")

	done := make(chan error, 1)
	go func() {
		_, err := manager.Attach("cache", spec)
		done <- err
	}()
	<-backend.attachStarted

	assert.Empty(t, manager.List())
	_, err := manager.Attach("cache", spec)
	assert.ErrorIs(t, err, ErrStorageDeviceBusy)
	_, err = manager.Detach("cache")
	assert.ErrorIs(t, err, ErrStorageDeviceBusy)

	close(backend.attachBlock)
	require.NoError(t, <-done)
	assert.Equal(t, []HotpluggedStorageDevice{
		{ID: "cache", Backend: "nbd", State: StorageDeviceStateConnecting},
	}, manager.List())

	_, err = manager.Detach("cache")
	require.NoError(t, err)
}

func TestStorageHotplugManagerTracksNBDConnectionState(t *testing.T) {
	backend := &fakeStorageHotplugBackend{}
	manager := newStorageHotplugManager(backend)
	_, err := manager.Attach("cache", testNBDStorageDevice("nbd://192.0.2.10:10809/export"))
	require.NoError(t, err)
	handle := backend.handles[0]
	stateOf := func() string { return manager.List()[0].State }

	assert.Equal(t, StorageDeviceStateConnecting, stateOf())
	handle.connected <- struct{}{}
	require.Eventually(t, func() bool { return stateOf() == StorageDeviceStateConnected }, time.Second, 10*time.Millisecond)
	handle.errCh <- errors.New("nbd://192.0.2.10:10809/export went away")
	require.Eventually(t, func() bool { return stateOf() == StorageDeviceStateDisconnected }, time.Second, 10*time.Millisecond)

	_, err = manager.Detach("cache")
	require.NoError(t, err)
}

func TestStorageHotplugManagerListIsSortedAndDoesNotExposeURI(t *testing.T) {
	backend := &fakeStorageHotplugBackend{}
	manager := newStorageHotplugManager(backend)
	_, err := manager.Attach("zeta", testNBDStorageDevice("nbd://192.0.2.10:10809/secret-z"))
	require.NoError(t, err)
	readOnly := testNBDStorageDevice("nbd://192.0.2.10:10809/secret-a")
	readOnly.ReadOnly = true
	_, err = manager.Attach("alpha", readOnly)
	require.NoError(t, err)

	assert.Equal(t, []HotpluggedStorageDevice{
		{ID: "alpha", Backend: "nbd", ReadOnly: true, State: StorageDeviceStateConnecting},
		{ID: "zeta", Backend: "nbd", ReadOnly: false, State: StorageDeviceStateConnecting},
	}, manager.List())

	_, err = manager.Detach("alpha")
	require.NoError(t, err)
	_, err = manager.Detach("zeta")
	require.NoError(t, err)
}

func TestStorageHotplugManagerDetachMissingIsIdempotent(t *testing.T) {
	manager := newStorageHotplugManager(&fakeStorageHotplugBackend{})

	detached, err := manager.Detach("missing")
	require.NoError(t, err)
	assert.False(t, detached)
}

func TestStorageHotplugManagerAttachesRawImage(t *testing.T) {
	backend := &fakeStorageHotplugBackend{}
	manager := newStorageHotplugManager(backend)
	spec := RawStorageDevice{Path: "/var/tmp/scratch.raw", ReadOnly: true}

	created, err := manager.AttachRaw("scratch", spec)
	require.NoError(t, err)
	assert.True(t, created)
	created, err = manager.AttachRaw("scratch", spec)
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, []RawStorageDevice{spec}, backend.rawCalls)
	assert.Equal(t, []HotpluggedStorageDevice{{
		ID:       "scratch",
		Backend:  "raw",
		ReadOnly: true,
		State:    StorageDeviceStateAttached,
	}}, manager.List())

	_, err = manager.Detach("scratch")
	require.NoError(t, err)
}

func TestValidateRawStorageDevice(t *testing.T) {
	dir := t.TempDir()

	err := validateRawStorageDevice(dir + "/missing.raw")
	assert.ErrorIs(t, err, ErrInvalidStorageDevice)
	err = validateRawStorageDevice(dir)
	assert.ErrorIs(t, err, ErrInvalidStorageDevice)
	assert.Contains(t, err.Error(), "not a regular file")

	odd := dir + "/odd.raw"
	require.NoError(t, writeFileOfSize(odd, 1000))
	err = validateRawStorageDevice(odd)
	assert.ErrorIs(t, err, ErrInvalidStorageDevice)
	assert.Contains(t, err.Error(), "multiple of 512")

	good := dir + "/good.raw"
	require.NoError(t, writeFileOfSize(good, 4096))
	assert.NoError(t, validateRawStorageDevice(good))
}

func TestRedactLocation(t *testing.T) {
	assert.Equal(t, "open <redacted>: denied", RedactLocation("open /var/tmp/x.raw: denied", "/var/tmp/x.raw"))
	assert.Equal(t, "unchanged", RedactLocation("unchanged", ""))
}

func writeFileOfSize(path string, size int) error {
	return os.WriteFile(path, make([]byte, size), 0600)
}
