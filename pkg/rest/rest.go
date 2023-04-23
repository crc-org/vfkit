package rest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/crc-org/vfkit/pkg/cmdline"
	"github.com/crc-org/vfkit/pkg/rest/define"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// VFKitService is used for the restful service; it describes
// the variables of the service like host/path but also has
// the router object
type VFKitService struct {
	Host   string
	Path   string
	Port   int
	router *gin.Engine
	Scheme ServiceScheme
}

// Start initiates the already configured gin service
func (v *VFKitService) Start() {
	go func() {
		var err error
		switch v.Scheme {
		case Tcp:
			err = v.router.Run(v.Host)
		case Unix:
			err = v.router.RunUnix(v.Path)
		}
		logrus.Fatal(err)
	}()
}

// NewServer creates a new restful service
func NewServer(vm *VzVirtualMachine, endpoint string) (*VFKitService, error) {
	uri, err := ParseRestfulURI(endpoint)
	if err != nil {
		return nil, err
	}
	scheme, err := ToRestScheme(uri.Scheme)
	if err != nil {
		return nil, err
	}
	r := gin.Default()
	s := VFKitService{
		router: r,
		Host:   uri.Host,
		Path:   uri.Path,
		Scheme: scheme,
	}

	// Handlers for the restful service.  This is where endpoints are defined.
	r.GET("/vm/state", vm.getVMState)
	r.POST("/vm/state", vm.setVMState)
	r.GET("/vm/inspect", vm.inspect)
	return &s, nil
}

// inspect returns information about the virtual machine like hw resources
// and devices
func (vm *VzVirtualMachine) inspect(c *gin.Context) {
	ii := define.InspectResponse{
		// TODO complete me
		CPUs:   1,
		Memory: 2048,
		//Devices: vm.Devices,
	}
	c.JSON(http.StatusOK, ii)
}

// getVMState retrieves the current vm state
func (vm *VzVirtualMachine) getVMState(c *gin.Context) {
	current := vm.GetState()
	c.JSON(http.StatusOK, gin.H{"state": current.String()})
}

// setVMState requests a state change on a virtual machine.  At this time only
// the following states are valid:
// Pause - pause a running machine
// Resume - resume a paused machine
// Stop - stops a running machine
// HardStop - forceably stops a running machine
func (vm *VzVirtualMachine) setVMState(c *gin.Context) {
	var (
		response error
		s        define.StateChangeRequest
	)

	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch s.NewState {
	case define.Pause:
		response = vm.Pause()
	case define.Resume:
		response = vm.Resume()
	case define.Stop:
		response = vm.Stop()
	case define.HardStop:
		response = vm.HardStop()
	default:
		eMsg := fmt.Errorf("invalid new StateResponse: %s", s.NewState)
		logrus.Error(eMsg)
		c.JSON(http.StatusBadRequest, gin.H{"error": eMsg.Error()})
		return

	}
	if response != nil {
		logrus.Errorf("failed action %s: %q", s.NewState, response)
		c.JSON(http.StatusInternalServerError, gin.H{"error": response.Error()})
		return
	}
	c.Status(http.StatusAccepted)
}

// ParseRestfulURI validates the input URI and returns an URL object
func ParseRestfulURI(inputURI string) (*url.URL, error) {
	restURI, err := url.ParseRequestURI(inputURI)
	if err != nil {
		return nil, err
	}
	scheme, err := ToRestScheme(restURI.Scheme)
	if err != nil {
		return nil, err
	}
	if scheme == Tcp && len(restURI.Host) < 1 {
		return nil, errors.New("invalid TCP uri: missing host")
	}
	if scheme == Tcp && len(restURI.Path) > 0 {
		return nil, errors.New("invalid TCP uri: path is forbidden")
	}
	if scheme == Tcp && restURI.Port() == "" {
		return nil, errors.New("invalid TCP uri: missing port")
	}
	if scheme == Unix && len(restURI.Path) < 1 {
		return nil, errors.New("invalid unix uri: missing path")
	}
	if scheme == Unix && len(restURI.Host) > 0 {
		return nil, errors.New("invalid unix uri: host is forbidden")
	}
	return restURI, err
}

// ToRestScheme converts a string to a ServiceScheme
func ToRestScheme(s string) (ServiceScheme, error) {
	switch strings.ToUpper(s) {
	case "NONE":
		return None, nil
	case "UNIX":
		return Unix, nil
	case "TCP":
		return Tcp, nil
	}
	return None, fmt.Errorf("invalid scheme %s", s)
}

func validateRestfulURI(inputURI string) error {
	if inputURI != cmdline.DefaultRestfulURI {
		if _, err := ParseRestfulURI(inputURI); err != nil {
			return err
		}
	}
	return nil
}
