package define

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

// SharedDir is one host directory exposed by a virtio-fs device.
type SharedDir struct {
	SharedDir string `json:"sharedDir"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

// VirtioFsShare replaces the directories exposed by a running virtio-fs device. Set either
// SharedDir (a single directory at the root of the mount) or Shares (named directories, each
// one a subdirectory of the mount; an empty map exposes nothing).
type VirtioFsShare struct {
	SharedDir string               `json:"sharedDir,omitempty"`
	ReadOnly  bool                 `json:"readOnly,omitempty"`
	Shares    map[string]SharedDir `json:"shares,omitempty"`
}
