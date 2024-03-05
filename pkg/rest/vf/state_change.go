package rest

import (
	"fmt"

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
		logrus.Debug("pausing virtual machine")
		response = vm.Pause()
	case define.Resume:
		logrus.Debug("resuming machine")
		response = vm.Resume()
	case define.Stop:
		logrus.Debug("stopping machine")
		_, response = vm.RequestStop()
	case define.HardStop:
		logrus.Debug("force stopping machine")
		response = vm.Stop()
	default:
		return fmt.Errorf("invalid new VMState: %s", newState)
	}
	return response
}
