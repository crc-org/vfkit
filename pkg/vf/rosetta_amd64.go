package vf

import (
	"fmt"
)

func (dev *RosettaShare) AddToVirtualMachineConfig(_ *vzVirtualMachineConfiguration) error {
	return fmt.Errorf("rosetta is unsupported on non-arm64 platforms")
}
