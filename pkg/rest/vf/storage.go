package rest

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"time"

	"github.com/crc-org/vfkit/pkg/config"
	corevf "github.com/crc-org/vfkit/pkg/vf"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	maximumStorageTimeoutMilliseconds int64 = 600_000
	maximumStorageLocationLength            = 4096
	maximumStorageRequestBytes        int64 = 64 << 10
)

// storageDeviceIDPattern matches the identifiers documented in doc/usage.md: a
// leading letter or digit followed by letters, digits, '.', '_' or '-'.
var storageDeviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,254}$`)

type storageDeviceRequest struct {
	Backend             string `json:"backend"`
	URI                 string `json:"uri"`
	Path                string `json:"path"`
	TimeoutMilliseconds int64  `json:"timeoutMilliseconds"`
	SynchronizationMode string `json:"synchronizationMode"`
	ReadOnly            bool   `json:"readOnly"`
}

func (vm *VzVirtualMachine) ListStorageDevices(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"devices": vm.storageController.HotpluggedStorageDevices()})
}

func (vm *VzVirtualMachine) AttachStorageDevice(c *gin.Context) {
	id := c.Param("id")
	if !storageDeviceIDPattern.MatchString(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage device ID"})
		return
	}

	request, err := decodeStorageDeviceRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage device request: " + err.Error()})
		return
	}
	created, err := vm.attachStorageDevice(id, request)
	if err != nil {
		location := request.sensitiveLocation()
		status, message := storageErrorResponse(err, "failed to attach storage device", location)
		if status == http.StatusInternalServerError {
			log.Errorf("Failed to attach storage device %s: %s", id, corevf.RedactLocation(err.Error(), location))
		}
		c.JSON(status, gin.H{"error": message})
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{"id": id})
}

// decodeStorageDeviceRequest reads the JSON body strictly: the body is capped,
// unknown fields are rejected so that misspelled options cannot silently fall
// back to defaults, and trailing content is refused.
func decodeStorageDeviceRequest(c *gin.Context) (storageDeviceRequest, error) {
	var request storageDeviceRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, maximumStorageRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return request, errors.New("unexpected content after the JSON object")
	}
	return request, nil
}

type storageRequestError struct {
	message string
}

func (err *storageRequestError) Error() string {
	return err.message
}

func invalidStorageRequest(message string) error {
	return &storageRequestError{message: message}
}

// storageErrorResponse maps errors from the storage controller to an HTTP
// status and a client-safe message; location is redacted from messages.
func storageErrorResponse(err error, fallback string, location string) (int, string) {
	var requestError *storageRequestError
	switch {
	case errors.As(err, &requestError):
		return http.StatusBadRequest, requestError.Error()
	case errors.Is(err, corevf.ErrInvalidStorageDevice):
		return http.StatusBadRequest, corevf.RedactLocation(err.Error(), location)
	case errors.Is(err, corevf.ErrStorageDeviceConflict), errors.Is(err, corevf.ErrStorageDeviceBusy):
		return http.StatusConflict, err.Error()
	case errors.Is(err, corevf.ErrUSBControllerUnavailable):
		return http.StatusServiceUnavailable, err.Error()
	default:
		return http.StatusInternalServerError, fallback
	}
}

func (vm *VzVirtualMachine) attachStorageDevice(id string, request storageDeviceRequest) (bool, error) {
	switch request.Backend {
	case "nbd":
		spec, err := request.nbdStorageDevice()
		if err != nil {
			return false, err
		}
		return vm.storageController.HotplugNBDStorageDevice(id, spec)
	case "raw":
		spec, err := request.rawStorageDevice()
		if err != nil {
			return false, err
		}
		return vm.storageController.HotplugRawStorageDevice(id, spec)
	default:
		return false, invalidStorageRequest("storage backend must be nbd or raw")
	}
}

func (vm *VzVirtualMachine) DetachStorageDevice(c *gin.Context) {
	id := c.Param("id")
	if !storageDeviceIDPattern.MatchString(id) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid storage device ID"})
		return
	}
	if _, err := vm.storageController.DetachHotpluggedStorageDevice(id); err != nil {
		status, message := storageErrorResponse(err, "failed to detach storage device", "")
		if status == http.StatusInternalServerError {
			log.Errorf("Failed to detach storage device %s: %v", id, err)
		}
		c.JSON(status, gin.H{"error": message})
		return
	}
	c.Status(http.StatusNoContent)
}

func (request storageDeviceRequest) nbdStorageDevice() (corevf.NBDStorageDevice, error) {
	if request.Path != "" {
		return corevf.NBDStorageDevice{}, invalidStorageRequest("path is not valid for the nbd backend")
	}
	if request.URI == "" {
		return corevf.NBDStorageDevice{}, invalidStorageRequest("storage URI is required")
	}
	if len(request.URI) > maximumStorageLocationLength {
		return corevf.NBDStorageDevice{}, invalidStorageRequest("storage URI is too long")
	}
	if err := config.ValidateNBDURI(request.URI); err != nil {
		return corevf.NBDStorageDevice{}, invalidStorageRequest("invalid storage URI: " + err.Error())
	}
	timeout := request.TimeoutMilliseconds
	if timeout == 0 {
		timeout = config.DefaultNBDTimeout.Milliseconds()
	}
	if timeout < 1 || timeout > maximumStorageTimeoutMilliseconds {
		return corevf.NBDStorageDevice{}, invalidStorageRequest("timeoutMilliseconds must be between 1 and 600000")
	}
	synchronizationMode := config.SynchronizationFullMode
	if request.SynchronizationMode != "" {
		mode, err := config.ParseNBDSynchronizationMode(request.SynchronizationMode)
		if err != nil {
			return corevf.NBDStorageDevice{}, invalidStorageRequest("synchronizationMode must be full or none")
		}
		synchronizationMode = mode
	}
	return corevf.NBDStorageDevice{
		URI:                 request.URI,
		Timeout:             time.Duration(timeout) * time.Millisecond,
		SynchronizationMode: synchronizationMode,
		ReadOnly:            request.ReadOnly,
	}, nil
}

func (request storageDeviceRequest) rawStorageDevice() (corevf.RawStorageDevice, error) {
	if request.URI != "" || request.TimeoutMilliseconds != 0 || request.SynchronizationMode != "" {
		return corevf.RawStorageDevice{}, invalidStorageRequest(
			"uri, timeoutMilliseconds, and synchronizationMode are not valid for the raw backend",
		)
	}
	if request.Path == "" {
		return corevf.RawStorageDevice{}, invalidStorageRequest("storage path is required")
	}
	if len(request.Path) > maximumStorageLocationLength {
		return corevf.RawStorageDevice{}, invalidStorageRequest("storage path is too long")
	}
	if !filepath.IsAbs(request.Path) {
		return corevf.RawStorageDevice{}, invalidStorageRequest("storage path must be absolute")
	}
	return corevf.RawStorageDevice{Path: request.Path, ReadOnly: request.ReadOnly}, nil
}

func (request storageDeviceRequest) sensitiveLocation() string {
	if request.Backend == "raw" {
		return request.Path
	}
	return request.URI
}
