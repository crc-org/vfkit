package rest

import (
	"net/http"

	"github.com/crc-org/vfkit/pkg/rest/define"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// VFKitService is used for the restful service; it describes
// the variables of the service like host/path but also has
// the router object
type VFKitService struct {
	*define.Endpoint
	router *gin.Engine
}

// Start initiates the already configured gin service
func (v *VFKitService) Start() {
	go func() {
		var err error
		switch v.Scheme {
		case define.TCP:
			err = v.router.Run(v.Host)
		case define.Unix:
			err = v.router.RunUnix(v.Path)
		}
		logrus.Fatal(err)
	}()
}

// NewServer creates a new restful service
func NewServer(vm *VzVirtualMachine, endpoint string) (*VFKitService, error) {
	r := gin.Default()
	ep, err := define.NewEndpoint(endpoint)
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
