package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/crc-org/vfkit/pkg/config"
	corevf "github.com/crc-org/vfkit/pkg/vf"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStorageController struct {
	devices       []corevf.HotpluggedStorageDevice
	attachCreated bool
	attachErr     error
	detachErr     error
	attachedID    string
	attachedSpec  corevf.NBDStorageDevice
	attachedRaw   corevf.RawStorageDevice
	detachedID    string
}

func (f *fakeStorageController) HotpluggedStorageDevices() []corevf.HotpluggedStorageDevice {
	return f.devices
}

func (f *fakeStorageController) HotplugNBDStorageDevice(
	id string,
	spec corevf.NBDStorageDevice,
) (bool, error) {
	f.attachedID = id
	f.attachedSpec = spec
	return f.attachCreated, f.attachErr
}

func (f *fakeStorageController) HotplugRawStorageDevice(
	id string,
	spec corevf.RawStorageDevice,
) (bool, error) {
	f.attachedID = id
	f.attachedRaw = spec
	return f.attachCreated, f.attachErr
}

func (f *fakeStorageController) DetachHotpluggedStorageDevice(id string) (bool, error) {
	f.detachedID = id
	return true, f.detachErr
}

// storageTestContext builds a gin context for method/path. body may be nil, a
// raw string, or a value that is JSON-encoded.
func storageTestContext(t *testing.T, method string, path string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	var requestBody bytes.Buffer
	switch b := body.(type) {
	case nil:
	case string:
		requestBody.WriteString(b)
	default:
		require.NoError(t, json.NewEncoder(&requestBody).Encode(body))
	}
	context.Request = httptest.NewRequest(method, path, &requestBody)
	return context, recorder
}

// attachContext builds a PUT context whose :id parameter is id. The request
// path itself stays fixed so that ids which are not valid URL paths can be
// exercised.
func attachContext(t *testing.T, id string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	context, recorder := storageTestContext(t, http.MethodPut, "/vm/storage/device", body)
	context.Params = gin.Params{{Key: "id", Value: id}}
	return context, recorder
}

func TestAttachStorageDevice(t *testing.T) {
	controller := &fakeStorageController{attachCreated: true}
	vm := &VzVirtualMachine{storageController: controller}
	context, recorder := attachContext(t, "avrea-persistent", map[string]any{
		"backend":             "nbd",
		"uri":                 "nbd://192.0.2.10:10809/export-token",
		"timeoutMilliseconds": 60_000,
		"synchronizationMode": "full",
		"readOnly":            false,
	})

	vm.AttachStorageDevice(context)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "avrea-persistent", controller.attachedID)
	assert.Equal(t, "nbd://192.0.2.10:10809/export-token", controller.attachedSpec.URI)
	assert.Equal(t, 60*time.Second, controller.attachedSpec.Timeout)
	assert.Equal(t, config.SynchronizationFullMode, controller.attachedSpec.SynchronizationMode)
	assert.False(t, controller.attachedSpec.ReadOnly)
}

func TestAttachStorageDeviceAppliesDefaults(t *testing.T) {
	controller := &fakeStorageController{attachCreated: true}
	vm := &VzVirtualMachine{storageController: controller}
	context, recorder := attachContext(t, "cache", map[string]any{
		"backend": "nbd",
		"uri":     "nbd://192.0.2.10:10809/export",
	})

	vm.AttachStorageDevice(context)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, config.DefaultNBDTimeout, controller.attachedSpec.Timeout)
	assert.Equal(t, config.SynchronizationFullMode, controller.attachedSpec.SynchronizationMode)
}

func TestAttachRawStorageDevice(t *testing.T) {
	controller := &fakeStorageController{attachCreated: true}
	vm := &VzVirtualMachine{storageController: controller}
	context, recorder := attachContext(t, "scratch", map[string]any{
		"backend":  "raw",
		"path":     "/var/tmp/scratch.raw",
		"readOnly": true,
	})

	vm.AttachStorageDevice(context)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Equal(t, "scratch", controller.attachedID)
	assert.Equal(t, corevf.RawStorageDevice{
		Path:     "/var/tmp/scratch.raw",
		ReadOnly: true,
	}, controller.attachedRaw)
}

func TestAttachRawStorageDeviceRejectsRelativePath(t *testing.T) {
	controller := &fakeStorageController{}
	vm := &VzVirtualMachine{storageController: controller}
	context, recorder := attachContext(t, "scratch", map[string]any{
		"backend": "raw",
		"path":    "scratch.raw",
	})

	vm.AttachStorageDevice(context)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Empty(t, controller.attachedID)
}

func TestAttachRawStorageDeviceDoesNotEchoPath(t *testing.T) {
	opaquePath := "/var/tmp/opaque-tenant-path/scratch.raw"
	controller := &fakeStorageController{attachErr: errors.New("could not open " + opaquePath)}
	vm := &VzVirtualMachine{storageController: controller}
	context, recorder := attachContext(t, "scratch", map[string]any{
		"backend": "raw",
		"path":    opaquePath,
	})

	vm.AttachStorageDevice(context)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), opaquePath)
}

func TestAttachRawStorageDeviceReportsInvalidImageAsBadRequest(t *testing.T) {
	opaquePath := "/var/tmp/opaque-tenant-path/scratch.raw"
	controller := &fakeStorageController{attachErr: fmt.Errorf(
		"%w: could not inspect raw disk image: stat %s: no such file or directory",
		corevf.ErrInvalidStorageDevice, opaquePath,
	)}
	vm := &VzVirtualMachine{storageController: controller}
	context, recorder := attachContext(t, "scratch", map[string]any{
		"backend": "raw",
		"path":    opaquePath,
	})

	vm.AttachStorageDevice(context)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "could not inspect raw disk image")
	assert.NotContains(t, recorder.Body.String(), opaquePath)
}

func TestAttachStorageDeviceDoesNotEchoURI(t *testing.T) {
	controller := &fakeStorageController{attachErr: errors.New("attach failed")}
	vm := &VzVirtualMachine{storageController: controller}
	opaqueURI := "nbd://192.0.2.10:10809/opaque-export"
	context, recorder := attachContext(t, "cache", map[string]any{
		"backend": "nbd",
		"uri":     opaqueURI,
	})

	vm.AttachStorageDevice(context)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), opaqueURI)
}

func TestAttachStorageDeviceRejectsInvalidID(t *testing.T) {
	for _, id := range []string{"bad id", "_cache", ".scratch", "-x", ""} {
		controller := &fakeStorageController{}
		vm := &VzVirtualMachine{storageController: controller}
		context, recorder := attachContext(t, id, map[string]any{
			"backend": "nbd",
			"uri":     "nbd://192.0.2.10:10809/export-token",
		})

		vm.AttachStorageDevice(context)

		assert.Equal(t, http.StatusBadRequest, recorder.Code, id)
		assert.Empty(t, controller.attachedID, id)
	}
}

func TestAttachStorageDeviceRejectsInvalidNBDURI(t *testing.T) {
	for _, uri := range []string{
		"https://example.com/secret-token",
		"nbd://",
		"nbd:relative",
		"nbd://host:0/x",
		"nbd://ho\"st/x",
		"nbd://ho<st>/x",
	} {
		controller := &fakeStorageController{}
		vm := &VzVirtualMachine{storageController: controller}
		context, recorder := attachContext(t, "cache", map[string]any{
			"backend": "nbd",
			"uri":     uri,
		})

		vm.AttachStorageDevice(context)

		assert.Equal(t, http.StatusBadRequest, recorder.Code, uri)
		assert.Empty(t, controller.attachedID, uri)
	}
}

func TestAttachStorageDeviceRejectsUnknownFields(t *testing.T) {
	controller := &fakeStorageController{}
	vm := &VzVirtualMachine{storageController: controller}
	context, recorder := attachContext(t, "scratch", map[string]any{
		"backend":   "raw",
		"path":      "/var/tmp/scratch.raw",
		"read_only": true,
	})

	vm.AttachStorageDevice(context)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "read_only")
	assert.Empty(t, controller.attachedID)
}

func TestAttachStorageDeviceRejectsTrailingContent(t *testing.T) {
	controller := &fakeStorageController{}
	vm := &VzVirtualMachine{storageController: controller}
	context, recorder := attachContext(t, "scratch",
		`{"backend":"raw","path":"/var/tmp/scratch.raw"}{"backend":"raw","path":"/var/tmp/other.raw"}`)

	vm.AttachStorageDevice(context)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Empty(t, controller.attachedID)
}

func TestAttachStorageDeviceRejectsOversizedBody(t *testing.T) {
	controller := &fakeStorageController{}
	vm := &VzVirtualMachine{storageController: controller}
	context, recorder := attachContext(t, "scratch", map[string]any{
		"backend": "raw",
		"path":    "/" + strings.Repeat("a", int(maximumStorageRequestBytes)),
	})

	vm.AttachStorageDevice(context)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Empty(t, controller.attachedID)
}

func TestAttachStorageDeviceConflict(t *testing.T) {
	for _, attachErr := range []error{corevf.ErrStorageDeviceConflict, corevf.ErrStorageDeviceBusy} {
		controller := &fakeStorageController{attachErr: attachErr}
		vm := &VzVirtualMachine{storageController: controller}
		context, recorder := attachContext(t, "cache", map[string]any{
			"backend": "nbd",
			"uri":     "nbd://192.0.2.10:10809/export-token",
		})

		vm.AttachStorageDevice(context)

		assert.Equal(t, http.StatusConflict, recorder.Code, attachErr)
	}
}

func TestAttachStorageDeviceWithoutController(t *testing.T) {
	controller := &fakeStorageController{attachErr: corevf.ErrUSBControllerUnavailable}
	vm := &VzVirtualMachine{storageController: controller}
	context, recorder := attachContext(t, "cache", map[string]any{
		"backend": "nbd",
		"uri":     "nbd://192.0.2.10:10809/export-token",
	})

	vm.AttachStorageDevice(context)

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestListStorageDevicesOmitsURI(t *testing.T) {
	controller := &fakeStorageController{devices: []corevf.HotpluggedStorageDevice{
		{ID: "cache", Backend: "nbd", ReadOnly: false, State: corevf.StorageDeviceStateConnected},
	}}
	vm := &VzVirtualMachine{storageController: controller}
	context, recorder := storageTestContext(t, http.MethodGet, "/vm/storage", nil)

	vm.ListStorageDevices(context)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.JSONEq(t, `{"devices":[{"id":"cache","backend":"nbd","readOnly":false,"state":"connected"}]}`, recorder.Body.String())
	assert.NotContains(t, recorder.Body.String(), "uri")
}

func TestDetachStorageDeviceIsIdempotent(t *testing.T) {
	controller := &fakeStorageController{}
	vm := &VzVirtualMachine{storageController: controller}
	context, _ := storageTestContext(t, http.MethodDelete, "/vm/storage/cache", nil)
	context.Params = gin.Params{{Key: "id", Value: "cache"}}

	vm.DetachStorageDevice(context)

	assert.Equal(t, http.StatusNoContent, context.Writer.Status())
	assert.Equal(t, "cache", controller.detachedID)
}

func TestDetachStorageDeviceBusy(t *testing.T) {
	controller := &fakeStorageController{detachErr: corevf.ErrStorageDeviceBusy}
	vm := &VzVirtualMachine{storageController: controller}
	context, recorder := storageTestContext(t, http.MethodDelete, "/vm/storage/cache", nil)
	context.Params = gin.Params{{Key: "id", Value: "cache"}}

	vm.DetachStorageDevice(context)

	assert.Equal(t, http.StatusConflict, recorder.Code)
}
