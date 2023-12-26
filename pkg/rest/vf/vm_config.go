package rest

import (
	"net/http"
	"sync"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/crc-org/vfkit/pkg/rest/define"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type VzVirtualMachine struct {
	VzVM     *vz.VirtualMachine
	config   *vz.VirtualMachineConfiguration
	vmConfig *config.VirtualMachine
}

func NewVzVirtualMachine(vm *vz.VirtualMachine, config *vz.VirtualMachineConfiguration, vmConfig *config.VirtualMachine) *VzVirtualMachine {
	return &VzVirtualMachine{config: config, VzVM: vm, vmConfig: vmConfig}
}

var (
	once            sync.Once
	devicesResponse define.DevicesResponse
)

func devicesToResp(devices []config.VirtioDevice) define.DevicesResponse {
	once.Do(func() {
		for _, dev := range devices {
			switch d := dev.(type) {
			case *config.USBMassStorage:
				devicesResponse.USBMassStorage = append(devicesResponse.USBMassStorage, *d)
			case *config.VirtioBlk:
				devicesResponse.Blk = append(devicesResponse.Blk, *d)
			case *config.RosettaShare:
				devicesResponse.Rosetta = *d
			case *config.NVMExpressController:
				devicesResponse.NVMe = append(devicesResponse.NVMe, *d)
			case *config.VirtioFs:
				devicesResponse.FS = append(devicesResponse.FS, *d)
			case *config.VirtioNet:
				n := define.VirtioNetResponse{
					Nat:            d.Nat,
					MacAddress:     d.MacAddress.String(),
					UnixSocketPath: d.UnixSocketPath,
				}

				if d.Socket != nil {
					n.Fd = int(d.Socket.Fd())
				}

				devicesResponse.Net = append(devicesResponse.Net, n)
			case *config.VirtioRng:
				devicesResponse.Rng = true
			case *config.VirtioSerial:
				devicesResponse.Serial = *d
			case *config.VirtioVsock:
				devicesResponse.Vsock = append(devicesResponse.Vsock, *d)
			case *config.VirtioInput:
				devicesResponse.Input = append(devicesResponse.Input, *d)
			case *config.VirtioGPU:
				devicesResponse.GPU = append(devicesResponse.GPU, *d)
			}
		}
	})

	return devicesResponse
}

// Inspect returns information about the virtual machine like hw resources
// and devices
func (vm *VzVirtualMachine) Inspect(c *gin.Context) {
	ii := define.InspectResponse{
		CPUs:    vm.vmConfig.Vcpus,
		Memory:  vm.vmConfig.MemoryBytes,
		Devices: devicesToResp(vm.vmConfig.Devices),
	}
	c.JSON(http.StatusOK, ii)
}

// GetVMState retrieves the current vm state
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

// SetVMState requests a state change on a virtual machine.  At this time only
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
