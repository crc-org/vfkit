package rest

import (
	"fmt"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/rest/define"
	"github.com/sirupsen/logrus"
)

// ChangeState execute a state change (i.e. running to stopped)
func (vm *VzVirtualMachine) ChangeState(newState define.StateChange) error {
	var (
		response error
	)
	switch newState {
	case define.Pause:
		response = vm.Pause()
	case define.Resume:
		response = vm.Resume()
	case define.Stop:
		response = vm.Stop()
	case define.HardStop:
		response = vm.HardStop()
	default:
		logrus.Error(response)
		return fmt.Errorf("invalid new VMState: %s", newState)
	}
	return response
}

// GetState returns state of the VM
func (vm *VzVirtualMachine) GetState() vz.VirtualMachineState {
	return vm.VzVM.State()
}

func (vm *VzVirtualMachine) Pause() error {
	logrus.Debug("pausing virtual machine")
	return vm.VzVM.Pause()
}

func (vm *VzVirtualMachine) Resume() error {
	logrus.Debug("resuming machine")
	return vm.VzVM.Resume()
}

func (vm *VzVirtualMachine) Stop() error {
	logrus.Debug("stopping machine")
	_, err := vm.VzVM.RequestStop()
	return err
}
func (vm *VzVirtualMachine) HardStop() error {
	logrus.Debug("force stopping machine")
	return vm.VzVM.Stop()
}

func (vm *VzVirtualMachine) CanStart() bool {
	return vm.VzVM.CanStart()
}

func (vm *VzVirtualMachine) CanPause() bool {
	return vm.VzVM.CanPause()
}

func (vm *VzVirtualMachine) CanResume() bool {
	return vm.VzVM.CanResume()
}

func (vm *VzVirtualMachine) CanStop() bool {
	return vm.VzVM.CanRequestStop()
}

func (vm *VzVirtualMachine) CanHardStop() bool {
	return vm.VzVM.CanStop()
}
