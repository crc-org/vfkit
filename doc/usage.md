# vfkit Command Line

The `vfkit` executable can be used to create a virtual machine (VM) using macOS virtualization framework.
The virtual machine will be terminated as soon as the `vfkit` process exits.
Its configuration can be specified through command line options.

Specifying VM bootloader configuration is mandatory.
Device configuration is optional, but most VM will need a disk image and a network interface to be configured.

## Generic Options
### Virtual Machine Resources

These options specify the amount of RAM and the number of CPUs which will be available to the virtual machine.
They are mandatory.

- `--cpus`

Number of virtual CPUs (vCPU) available in the VM. It defaults to 1 vCPU.

- `--memory`

Amount of memory available in the virtual machine. The value is in MiB (mibibytes, 1024 * 1024 * 1024 bytes), and the default is 512 MiB.

### Time Synchronization Configuration

#### Description

When the host system is suspended, the guest clock stops running, and it's unable to get back to the correct time upon resume.
The `--timesync` option can be used to let `vfkit` set the guest clock to the correct time when it detects the host.
At the moment, this can only be done using `qemu-guest-agent`, which has to be installed in the guest.
It must be configured to communicate over virtio-vsock.

#### Arguments
- `vsockPort`: vsock port used for communication with the guest agent.


## Bootloader Configuration

A bootloader is required to tell vfkit _how_ it should be starting the guest OS.

### Linux bootloader

#### Description

`--bootloader linux` replaces the legacy `--kernel`, `--kernel-cmdline` and `--initrd` options.
It allows to specify which kernel and initrd should be used when starting the VM.

#### Arguments

- `kernel`: path to the kernel to use to start the virtual machine. The kernel *must* be uncompressed or the VM will hang when trying to start. See [the kernel documentation](https://www.kernel.org/doc/Documentation/arm64/booting.txt) for more details.
- `initrd`: path to the initrd file to use when starting the virtual machine.
- `cmdline`: kernel command line to use when starting the virtual machine.

#### Example

`--bootloader linux,kernel=~/kernels/vmlinuz-5.18.18-200.fc36.aarch64,initrd=~/kernels/initramfs-5.18.18-200.fc36.aarch64.img,cmdline="\"console=hvc0 root=UUID=164b4fc3-dc5a-40ea-a40b-c689a7bf41cf rw\""`

The kernel command line must be enclosed in `"`, and depending on your shell, they might need to be escaped (`\"`)


### EFI bootloader

#### Description

`--bootloader efi` is only available when running on macOS 13 or newer.
This allows to boot a disk image using EFI, which removes the need for providing external kernel/initrd/...
The disk image bootloader will be started by the EFI firmware, which will in turn know which kernel it should be booting.

#### Arguments

- `variable-store: path to a file which EFI can use to store its variables
- `create`: indicate whether the `variable-store` file should be created or not if missing.

### Deprecated options

#### Description

The `--kernel`, `--initrd` and `--kernel-cmdline` options are deprecated and have been replaced by the more generic `--bootloader` option.

#### Options

- `--kernel`

Path to the kernel to use to start the virtual machine. The kernel *must* be uncompressed or the VM will hang when trying to start.
See [the kernel documentation](https://www.kernel.org/doc/Documentation/arm64/booting.txt) for more details.

- `--initrd`

Path to the initrd file to use when starting the virtual machine.

- `--kernel-cmdline`

Kernel command line to use when starting the virtual machine.


## Device Configuration

Various devices can be added to the virtual machines. They are all paravirtualized devices using VirtIO. They are grouped under the `--device` commande line flag.


### Disk

#### Description

The `--device virtio-blk` option adds a disk to the virtual machine. The disk is backed by an image file on the host machine. This file is a raw image file.
This means an empty 1GiB disk can be created with `dd if=/dev/zero of=vfkit.img bs=1G count=1`.
See also [vz/CreateDiskImage](https://pkg.go.dev/github.com/Code-Hex/vz/v3#CreateDiskImage).

#### Arguments
- `path`: the absolute path to the disk image file.
- `deviceId`: `/dev/disk/by-id/` identifier to use for this device.

#### Example
`--device virtio-blk,path=/Users/virtuser/vfkit.img`


### USB Mass Storage

#### Description

The `--device usb-mass-storage` option adds a USB mass storage device to the virtual machine. The disk is backed by an image file on the host machine. This file is a raw image file or an ISO image.

#### Arguments
- `path`: the absolute path to the disk image file.

#### Example
`--device usb-mass-storage,path=/Users/virtuser/distro.iso`


### Networking

#### Description

The `--device virtio-net` option adds a network interface to the virtual machine. If it gets its IP address through DHCP, its IP can be found in `/var/db/dhcpd_leases` on the host.

#### Arguments
- `mac`: optional argument to specify the MAC address of the VM. If it's omitted, a random MAC address will be used.

#### Example
`--device virtio-net,mac=52:54:00:70:2b:71`


### Serial Port

#### Description

The `--device virtio-serial` option adds a serial device to the virtual machine. This is useful to redirect text output from the virtual machine to a log file.

#### Arguments
- `logFilePath`: path where the serial port output should be written.

#### Example
`--device virtio-serial,logFilePath=/Users/virtuser/vfkit.log`


### Random Number Generator

#### Description

The `--device virtio-rng` option adds a random number generator device to the virtual machine.
It will feed entropy from the host to the virtual machine, as VMs often do not have many entropy sources.

#### Example
`--device virtio-rng`


### virtio-vsock communication

#### Description

The `--device virtio-vsock` option adds a virtio-vsock communication channel between the host and the guest
See `man 4 vsock` for more details. The vsock port will be exposed as a unix socket on the host. 

#### Arguments
- `port`: vsock port to use for the VM/host communication.
- `socketURL`: path to the unix socket to use on the host for the vsock communication.
- `connect`: indicates that the host will connect to the guest over vsock.
- `listen` : indicates that the host will be listening for vsock connections (default).

#### Example
`--device virtio-vsock,port=5,socketURL=/Users/virtuser/vfkit.sock`


### File Sharing

#### Description

The `-device virtio-fs` option allows to share directories between the host and the guest. The sharing will be done using virtio-fs.
The share can be mounted in the guest with `mount -t virtio-fs vfkitTag /mnt`, with `vfkitTag` corresponding to the value of the `mountTag` option.


#### Arguments
- `sharedDir`: absolute path to the host directory to share with the guest.
- `mountTag`: tag which will be used to mount the shared directory in the guest.

#### Example
`--device virtio-fs,sharedDir=/Users/virtuser/vfkit/,mountTag=vfkit-share`

