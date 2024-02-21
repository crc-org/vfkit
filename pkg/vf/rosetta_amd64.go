package vf

import (
	"fmt"
)

func (dev *RosettaShare) AddToVirtualMachineConfig(_ *VirtualMachineConfiguration) error {
	return fmt.Errorf("rosetta is unsupported on non-arm64 platforms")
}
