package rest

import (
	"github.com/Code-Hex/vz/v3"
)

type VzVirtualMachine struct {
	VzVM   *vz.VirtualMachine
	state  vz.VirtualMachineState
	config *vz.VirtualMachineConfiguration
}

func NewVzVirtualMachine(vm *vz.VirtualMachine, config *vz.VirtualMachineConfiguration) *VzVirtualMachine {
	return &VzVirtualMachine{config: config, VzVM: vm}
}
