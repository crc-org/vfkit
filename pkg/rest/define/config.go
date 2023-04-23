package define

// InspectResponse is used when responding to a request for
// information about the virtual machine
type InspectResponse struct {
	CPUs   uint   `json:"cpus"`
	Memory uint64 `json:"memory"`
	//Devices []config.VirtioDevice `json:"devices"`
}

// StateResponse is for responding to a request for virtual
// machine state
type StateResponse struct {
	State string `json:"state"`
}

// StateChangeRequest is used by the restful service consumer
// to ask for a virtual machine state change
type StateChangeRequest struct {
	NewState StateChange `json:"new_state"`
}

// StateChange is a string strong typing of values for changing
// the state of a virtual machine
type StateChange string

const (
	Resume   StateChange = "Resume"
	Pause    StateChange = "Pause"
	Stop     StateChange = "Stop"
	HardStop StateChange = "HardStop"
)
