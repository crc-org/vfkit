package rest

import (
	"net/http"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/rest/define"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type VzVirtualMachine struct {
	VzVM   *vz.VirtualMachine
	config *vz.VirtualMachineConfiguration
}

func NewVzVirtualMachine(vm *vz.VirtualMachine, config *vz.VirtualMachineConfiguration) *VzVirtualMachine {
	return &VzVirtualMachine{config: config, VzVM: vm}
}

// inspect returns information about the virtual machine like hw resources
// and devices
func (vm *VzVirtualMachine) Inspect(c *gin.Context) {
	ii := define.InspectResponse{
		// TODO complete me
		CPUs:   1,
		Memory: 2048,
		//Devices: vm.Devices,
	}
	c.JSON(http.StatusOK, ii)
}

// getVMState retrieves the current vm state
func (vm *VzVirtualMachine) GetVMState(c *gin.Context) {
	current := vm.GetState()
	c.JSON(http.StatusOK, gin.H{
		"state":       current.String(),
		"canStart":    vm.CanStart(),
		"canPause":    vm.CanPause(),
		"canResume":   vm.CanResume(),
		"canStop":     vm.CanStop(),
		"canHardStop": vm.CanHardStop(),
	})
}

// setVMState requests a state change on a virtual machine.  At this time only
// the following states are valid:
// Pause - pause a running machine
// Resume - resume a paused machine
// Stop - stops a running machine
// HardStop - forceably stops a running machine
func (vm *VzVirtualMachine) SetVMState(c *gin.Context) {
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
