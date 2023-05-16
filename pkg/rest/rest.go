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

type Endpoint struct {
	Host   string
	Path   string
	Scheme ServiceScheme
}

func NewEndpoint(input string) (*Endpoint, error) {
	uri, err := parseRestfulURI(input)
	if err != nil {
		return nil, err
	}
	scheme, err := toRestScheme(uri.Scheme)
	if err != nil {
		return nil, err
	}
	return &Endpoint{
		Host:   uri.Host,
		Path:   uri.Path,
		Scheme: scheme,
	}, nil
}

func (ep *Endpoint) ToCmdLine() ([]string, error) {
	args := []string{"--restful-uri"}
	switch ep.Scheme {
	case Unix:
		args = append(args, fmt.Sprintf("unix://%s", ep.Path))
	case TCP:
		args = append(args, fmt.Sprintf("tcp://%s%s", ep.Host, ep.Path))
	case None:
		return []string{}, nil
	default:
		return []string{}, errors.New("invalid endpoint scheme")
	}
	return args, nil
}

// VFKitService is used for the restful service; it describes
// the variables of the service like host/path but also has
// the router object
type VFKitService struct {
	*Endpoint
	router *gin.Engine
}

// Start initiates the already configured gin service
func (v *VFKitService) Start() {
	go func() {
		var err error
		switch v.Scheme {
		case TCP:
			err = v.router.Run(v.Host)
		case Unix:
			err = v.router.RunUnix(v.Path)
		}
		logrus.Fatal(err)
	}()
}

// NewServer creates a new restful service
func NewServer(vm *VzVirtualMachine, endpoint string) (*VFKitService, error) {
	r := gin.Default()
	ep, err := NewEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	s := VFKitService{
		router:   r,
		Endpoint: ep,
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
		s define.VMState
	)

	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := vm.ChangeState(define.StateChange(s.State))
	if response != nil {
		logrus.Errorf("failed action %s: %q", s.State, response)
		c.JSON(http.StatusInternalServerError, gin.H{"error": response.Error()})
		return
	}
	c.Status(http.StatusAccepted)
}

// parseRestfulURI validates the input URI and returns an URL object
func parseRestfulURI(inputURI string) (*url.URL, error) {
	restURI, err := url.ParseRequestURI(inputURI)
	if err != nil {
		return nil, err
	}
	scheme, err := toRestScheme(restURI.Scheme)
	if err != nil {
		return nil, err
	}
	if scheme == TCP && len(restURI.Host) < 1 {
		return nil, errors.New("invalid TCP uri: missing host")
	}
	if scheme == TCP && len(restURI.Path) > 0 {
		return nil, errors.New("invalid TCP uri: path is forbidden")
	}
	if scheme == TCP && restURI.Port() == "" {
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

// toRestScheme converts a string to a ServiceScheme
func toRestScheme(s string) (ServiceScheme, error) {
	switch strings.ToUpper(s) {
	case "NONE":
		return None, nil
	case "UNIX":
		return Unix, nil
	case "TCP", "HTTP":
		return TCP, nil
	}
	return None, fmt.Errorf("invalid scheme %s", s)
}

func validateRestfulURI(inputURI string) error {
	if inputURI != cmdline.DefaultRestfulURI {
		if _, err := parseRestfulURI(inputURI); err != nil {
			return err
		}
	}
	return nil
}
