# Missing APIs

This document contains a non-exhaustive list of APIs not currently used/supported by `vfkit`, but could be useful to have.

## non-vz APIs

- start vfkit process (integrating with https://pkg.go.dev/os/exec )
- get VM IP address [https://github.com/code-ready/crc/blob/0d76300c1a618598c209bab32a8deb4ca6c2d8c6/pkg/drivers/vfkit/network_darwin.go#L54-L59]

## [vz](https://pkg.go.dev/github.com/Code-Hex/vz/v3) APIs
```
    func vz.CreateDiskImage(pathname string, size int64) error
    func vz.VirtualMachineConfigurationMaximumAllowedCPUCount() uint
    func vz.VirtualMachineConfigurationMaximumAllowedMemorySize() uint64
    func vz.VirtualMachineConfigurationMinimumAllowedCPUCount() uint
    func vz.VirtualMachineConfigurationMinimumAllowedMemorySize() uint64

    type FileHandleNetworkDeviceAttachment
        func vz.NewFileHandleNetworkDeviceAttachment(file *os.File) *FileHandleNetworkDeviceAttachment
    type FileHandleSerialPortAttachment
        func vz.NewFileHandleSerialPortAttachment(read, write *os.File) *FileHandleSerialPortAttachment
    type MultipleDirectoryShare
        func vz.NewMultipleDirectoryShare(shares map[string]*SharedDirectory) *MultipleDirectoryShare
    type VirtioTraditionalMemoryBalloonDeviceConfiguration
        func vz.NewVirtioTraditionalMemoryBalloonDeviceConfiguration() *VirtioTraditionalMemoryBalloonDeviceConfiguration
    type VirtualMachine
        func (v *VirtualMachine) vz.CanPause() bool
        func (v *VirtualMachine) vz.CanRequestStop() bool
        func (v *VirtualMachine) vz.CanResume() bool
        func (v *VirtualMachine) vz.CanStart() bool
        func (v *VirtualMachine) vz.CanStop() bool
        func (v *VirtualMachine) vz.Pause(fn func(error))
        func (v *VirtualMachine) vz.RequestStop() (bool, error)
        func (v *VirtualMachine) vz.Resume(fn func(error))
        func (v *VirtualMachine) vz.State() VirtualMachineState
        func (v *VirtualMachine) vz.Stop(fn func(error))
    type VirtualMachineConfiguration
        func (v *VirtualMachineConfiguration) vz.SetMemoryBalloonDevicesVirtualMachineConfiguration(cs []MemoryBalloonDeviceConfiguration)
```
