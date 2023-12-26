package define

import (
	"github.com/crc-org/vfkit/pkg/config"
)

type VirtioNetResponse struct {
	Nat            bool   `json:"nat"`
	MacAddress     string `json:"macAddress"`
	UnixSocketPath string `json:"unixSocketPath"`
	Fd             int    `json:"fd"`
}

type DevicesResponse struct {
	Input          []config.VirtioInput          `json:"input"`
	GPU            []config.VirtioGPU            `json:"gpu"`
	Vsock          []config.VirtioVsock          `json:"vsock"`
	Blk            []config.VirtioBlk            `json:"blk"`
	FS             []config.VirtioFs             `json:"fs"`
	Rosetta        config.RosettaShare           `json:"rosetta"`
	NVMe           []config.NVMExpressController `json:"nvme"`
	Net            []VirtioNetResponse           `json:"net"`
	Rng            bool                          `json:"rng"`
	Serial         config.VirtioSerial           `json:"serial"`
	USBMassStorage []config.USBMassStorage       `json:"usbMassStorage"`
}

// InspectResponse is used when responding to a request for
// information about the virtual machine
type InspectResponse struct {
	CPUs    uint            `json:"cpus"`
	Memory  uint64          `json:"memory"`
	Devices DevicesResponse `json:"devices"`
}

// VMState can be used to describe the current state of a VM
// as well as used to request a state change
type VMState struct {
	State string `json:"state"`
}

type StateChange string

const (
	Resume   StateChange = "Resume"
	Pause    StateChange = "Pause"
	Stop     StateChange = "Stop"
	HardStop StateChange = "HardStop"
)
