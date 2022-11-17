// Package client provides a go API to generate a vfkit commandline.
//
// After creating a `VirtualMachine` object, use its `ToCmdLine()` method to
// get a list of arguments which can be used with the [os/exec] package.
// package client
package client

import (
	"net"
)

// Bootloader is the base interface for all bootloader classes. It specifies how to
// boot the virtual machine. It is mandatory to set a Bootloader or the virtual
// machine won't start.
type Bootloader VMComponent

// linuxBootloader determines which kernel/initrd/kernel args to use when starting
// the virtual machine.
type linuxBootloader struct {
	vmlinuzPath   string
	kernelCmdLine string
	initrdPath    string
}

// efiBootloader allows to set a few options related to EFI variable storage
type efiBootloader struct {
	efiVariableStorePath string
	createVariableStore  bool
}

// VirtualMachine is the top-level type. It describes the virtual machine
// configuration (bootloader, devices, ...).
type VirtualMachine struct {
	vcpus       uint
	memoryBytes uint64
	bootloader  Bootloader
	devices     []VirtioDevice
}

// The VMComponent interface represents a VM element (device, bootloader, ...)
// which can be converted to commandline parameters
type VMComponent interface {
	ToCmdLine() ([]string, error)
}

// The VirtioDevice interface is an interface which is implemented by all devices.
type VirtioDevice VMComponent

// VirtioVsock configures of a virtio-vsock device allowing 2-way communication
// between the host and the virtual machine type
type VirtioVsock struct {
	// Port is the virtio-vsock port used for this device, see `man vsock` for more
	// details.
	Port uint
	// SocketURL is the path to a unix socket on the host to use for the virtio-vsock communication with the guest.
	SocketURL string
	// If true, vsock connections will have to be done from guest to host. If false, vsock connections will only be possible
	// from host to guest
	Listen bool
}

// virtioBlk configures a disk device.
type virtioBlk struct {
	imagePath string
}

// virtioRNG configures a random number generator (RNG) device.
type virtioRNG struct {
}

// virtioNet configures the virtual machine networking.
type virtioNet struct {
	nat        bool
	macAddress net.HardwareAddr
}

// virtioSerial configures the virtual machine serial ports.
type virtioSerial struct {
	logFile string
}

// virtioFs configures directory sharing between the guest and the host.
type virtioFs struct {
	sharedDir string
	mountTag  string
}

// timeSync enables synchronization of the host time to the linux guest after the host was suspended.
// This requires qemu-guest-agent to be running in the guest, and to be listening on a vsock socket
type timeSync struct {
	vsockPort uint
}
