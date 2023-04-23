package rest

import (
	"errors"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/rest/define"
	"github.com/sirupsen/logrus"
)

// ErrNotImplemented Temporary Error Message
var ErrNotImplemented = errors.New("function not implemented yet")

// ChangeState execute a state change (i.e. running to stopped)
func (vm *VzVirtualMachine) ChangeState(newState define.StateChange) error {
	return ErrNotImplemented
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
