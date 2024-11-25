# vfkit Command Line

The `vfkit` executable can be used to create a virtual machine (VM) using macOS virtualization framework.
The virtual machine will be terminated as soon as the `vfkit` process exits.
Its configuration can be specified through command line options.

Specifying VM bootloader configuration is mandatory.
Device configuration is optional, but most VM will need a disk image and a network interface to be configured.

## Generic Options

- `--log-level`

Set the log-level for VFKit.  Supported values are `debug`, `info`, and `error`.

- `--restful-uri`

The URI (address) of the RESTful service.  The default is `tcp://localhost:8081`.  Valid schemes are
`tcp`, `none`, or `unix`.  In the case of unix, the "host" portion would be a path to where the unix domain socket will be stored. A scheme of `none` disables the RESTful service.

### Virtual Machine Resources

These options specify the amount of RAM and the number of CPUs which will be available to the virtual machine.
They are mandatory.

- `--cpus`

Number of virtual CPUs (vCPU) available in the VM. It defaults to 1 vCPU.

- `--memory`

Amount of memory available in the virtual machine. The value is in MiB ([mebibytes](https://simple.wikipedia.org/wiki/Mebibyte), 1024 * 1024 bytes), and the default is 512 MiB.

### Time Synchronization Configuration

#### Description

When the host system is suspended, the guest clock stops running, and it's unable to get back to the correct time upon resume.
The `--timesync` option can be used to let `vfkit` set the guest clock to the correct time when it detects the host.
At the moment, this can only be done using `qemu-guest-agent`, which has to be installed in the guest.
It must be configured to communicate over virtio-vsock.

#### Arguments
- `vsockPort`: vsock port used for communication with the guest agent.


## Bootloader Configuration

A bootloader is required to tell vfkit _how_ it should start the guest OS.

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

### macOS bootloader

#### Description

`--bootloader macos` is required to run macOS VMs. You must use an arm64/Apple silicon device running macOS 12 or later. Due to hardcoded limitations in the Apple Virtualization framework, it's not possible to run more than two macOS VMs at a time. Since macOS guests can't run headlessly, you'll need to enable a GUI, even if you only plan to interact with the VM over SSH.

#### Arguments

- `machineIdentifierPath`: absolute path to a binary property list containing a unique ECID identifier for the VM
- `hardwareModelPath`: absolute path to a binary property list defining OS version support
- `auxImagePath`: absolute path to the auxiliary storage file with NVRAM contents and the iBoot bootloader

#### Example

`--bootloader macos,machineIdentifierPath=/Users/virtuser/VM.bundle/MachineIdentifier,hardwareModelPath=/Users/virtuser/VM.bundle/HardwareModel,auxImagePath=/Users/virtuser/VM.bundle/AuxiliaryStorage`

### EFI bootloader

#### Description

`--bootloader efi` is only available when running on macOS 13 or newer.
This allows to boot a disk image using EFI, which removes the need for providing external kernel/initrd/...
The disk image bootloader will be started by the EFI firmware, which will in turn know which kernel it should be booting.

#### Arguments

- `variable-store`: path to a file which EFI can use to store its variables
- `create`: indicate whether the `variable-store` file should be created or not if missing.

#### Example

`--bootloader efi,variable-store=/Users/virtuser/efi-variable-store,create`


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

Various devices can be added to the virtual machines. They are all paravirtualized devices using VirtIO. They are grouped under the `--device` command line flag.


### Disk

#### Description

The `--device virtio-blk` option adds a disk to the virtual machine. The disk is backed by an image file on the host machine. This file is a raw image file.
See also [vz/CreateDiskImage](https://pkg.go.dev/github.com/Code-Hex/vz/v3#CreateDiskImage).

#### Thin images

Apple Virtualization Framework only supports raw disk images and ISO images.
There is no support for thin image formats such as [qcow2](https://en.wikipedia.org/wiki/Qcow).

However, APFS, the default macOS filesystem has support for sparse files and copy-on-write files, so it offers the main features of thin image formats.

A sparse raw image can be created/expanded using the `truncate` command or
using [`truncate(2)`](https://manpagez.com/man/2/truncate/).
For example, an empty 1GiB disk can be created with `truncate -s 1G
vfkit.img`. Such an image will only use disk space when content is written to
it. It initially only uses a few bytes of actual disk space even if its size
is 1G.

A copy-on-write image is a raw image file which references a backing file. Its
initial content is the same as its backing file, and the data is shared with
the backing file. This means the copy-on-write image does not use extra disk
space when it's created. When this image is modified, the changes will only be
made to the copy-on-write image, and not to the backing file. Only the
modified data will use actual disk space.
A copy-on-write image can be created using `cp -c` or [clonefile(2)](http://www.manpagez.com/man/2/clonefile/).

#### Cloud-init

The `--device virtio-blk` option can also be used to supply an initial configuration to cloud-init through a disk image.

The ISO image file must be labelled cidata or CIDATA and it must contain the user-data and meta-data files. 
It is also possible to add further configurations by using the network-config and vendor-data files.
See https://cloudinit.readthedocs.io/en/latest/reference/datasources/nocloud.html#runtime-configurations for more details.

To create the ISO image you can use the following command within a folder containing the user-data and meta-data files
```
mkisofs -output seed.img -volid cidata -rock {user-data,meta-data}
```

See https://cloudinit.readthedocs.io/en/latest/reference/datasources/nocloud.html#example-creating-a-disk for further details about how to create a disk image

#### Arguments
- `path`: the absolute path to the disk image file.
- `deviceId`: `/dev/disk/by-id/` identifier to use for this device.

#### Example

This adds a virtio-blk device to the VM which will be backed by the raw image at `/Users/virtuser/vfkit.img`:
```
--device virtio-blk,path=/Users/virtuser/vfkit.img
```

To also provide the cloud-init configuration you can add an additional virtio-blk device backed by an image containing the cloud-init configuration files
```
--device virtio-blk,path=/Users/virtuser/cloudinit.img
```


### NVM Express

#### Description

The `--device nvme` option adds a NVMe device to the virtual machine. The disk is backed by an image file on the host machine. This file is a raw image file.

#### Arguments
- `path`: the absolute path to the disk image file.

#### Example

This adds a NVMe device to the VM which will be backed by the disk image at `/Users/virtuser/image.img`:
```
--device nvme,path=/Users/virtuser/image.img
```


### USB Mass Storage

#### Description

The `--device usb-mass-storage` option adds a USB mass storage device to the virtual machine. The disk is backed by an image file on the host machine. This file is a raw image file or an ISO image.

#### Arguments
- `path`: the absolute path to the disk image file.
- `readonly`: if specified the device will be read only.

#### Example

This adds a read only USB mass storage device to the VM which will be backed by the ISO image at `/Users/virtuser/distro.iso`:
```
--device usb-mass-storage,path=/Users/virtuser/distro.iso,readonly
```

### Network Block Device

#### Description

The `--device nbd` option allows to connect to a remote NBD server, effectively accessing a remote block device over the network as if it were a local disk.

The NBD client running on the VM is informed in case the connection drops and it tries to reconnect automatically to the server.

#### Arguments
- `uri`: the URI that refers to the NBD server to which the NBD client will connect, e.g. `nbd://10.10.2.8:10000/export`. More info at https://github.com/NetworkBlockDevice/nbd/blob/master/doc/uri.md
- `deviceId`: `/dev/disk/by-id/virtio-` identifier to use for this device.
- `sync`: the mode in which the NBD client synchronizes data with the NBD server. It can be `full`or `none`, more info at https://developer.apple.com/documentation/virtualization/vzdisksynchronizationmode?language=objc
- `timeout`: the timeout value in milliseconds for the connection between the client and server
- `readonly`: if specified the device will be read only.

#### Example

This allows to connect to the export of the remote NBD server:
```
--device nbd,uri=nbd://192.168.64.4:11111/export,deviceId=nbd1,timeout=3000
```


### Networking

#### Description

The `--device virtio-net` option adds a network interface to the virtual machine. If it gets its IP address through DHCP, its IP can be found in `/var/db/dhcpd_leases` on the host.

#### Arguments
- `mac`: optional argument to specify the MAC address of the VM. If it's omitted, a random MAC address will be used.
- `fd`: file descriptor to attach to the guest network interface. The file descriptor must be a connected datagram socket. See [VZFileHandleNetworkDeviceAttachment](https://developer.apple.com/documentation/virtualization/vzfilehandlenetworkdeviceattachment?language=objc) for more details.
- `nat`: guest network traffic will be NAT'ed through the host. This is the default. See [VZNATNetworkDeviceAttachment](https://developer.apple.com/documentation/virtualization/vznatnetworkdeviceattachment?language=objc) for more details.
- `unixSocketPath`: path to a unix socket to attach to the guest network interface. See [VZFileHandleNetworkDeviceAttachment](https://developer.apple.com/documentation/virtualization/vzfilehandlenetworkdeviceattachment?language=objc) for more details.

`fd`, `nat`, `unixSocketPath` are mutually exclusive.

#### Example

This adds a virtio-net device to the VM with `52:54:00:70:2b:71` as its MAC address:
```
--device virtio-net,nat,mac=52:54:00:70:2b:71
```

This adds a virtio-net device to the VM, and redirects all the network traffic on the corresponding guest network interface to `/Users/virtuser/virtio-net.sock`:
```
--device virtio-net,unixSocketPath=/Users/virtuser/virtio-net.sock
```
This is useful in combination with usermode networking stacks such as [gvisor-tap-vsock](https://github.com/containers/gvisor-tap-vsock).


### Serial Port

#### Description

The `--device virtio-serial` option adds a serial device to the virtual machine. This is useful to redirect text output from the virtual machine to a log file.
The `logFilePath`, `stdio`, `pty` arguments are mutually exclusive.

#### Arguments
- `logFilePath`: path where the serial port output should be written.
- `stdio`: uses stdin/stdout for the serial console input/output.
- `pty`: allocates a pseudo-terminal for the serial console input/output.

#### Example

This adds a virtio-serial device to the VM, and will log everything which is written to this device to `/Users/virtuser/vfkit.log`:
```
--device virtio-serial,logFilePath=/Users/virtuser/vfkit.log
```

This adds a virtio-serial device to the VM, and the terminal `vfkit` is
launched from will be used as an interactive serial console for that device:
```
--device virtio-serial,stdio
```

This adds a virtio-serial device to the VM, and creates a pseudo-terminal for
the console for that device:
```
--device virtio-serial,pty
```
Once the VM is running, you can connect to its console with:
```
screen /dev/ttys002
```
`/dev/ttys002` will vary between `vfkit` runs.
The `/dev/ttys???` path to the pty is printed during vfkit startup.
It's also available through the `/vm/inspect` endpoint of [REST API](#restful-service) in the `ptyName` field of the `virtio-serial` device.

### Random Number Generator

#### Description

The `--device virtio-rng` option adds a random number generator device to the virtual machine.
It will feed entropy from the host to the virtual machine, as VMs often do not have many entropy sources.

#### Example

This adds a virtio-rng device to the VM:
```
--device virtio-rng
```


### virtio-vsock communication

#### Description

The `--device virtio-vsock` option adds a virtio-vsock communication channel between the host and the guest
See `man 4 vsock` for more details. macOS does not have host support for
`AF_VSOCK` sockets so the vsock port will be exposed as a unix socket on the
host.

`--device virtio-vsock` can be specified multiple times on the command line to
allow communication over multiple vsock ports. There will only be a single
virtio-vsock device added to the VM regardless of the number of `--device
virtio-vsock` occurrences on the command line.

#### Arguments
- `port`: vsock port to use for the VM/host communication.
- `socketURL`: path to the unix socket to use on the host for the vsock communication.
- `connect`: indicates that the host will connect to the guest over vsock.
- `listen` : indicates that the host will be listening for vsock connections (default).

#### Example

This allows virtio-vsock communication from the guest to the host over vsock port 5:
```
--device virtio-vsock,port=5,socketURL=/Users/virtuser/vfkit-5.sock
```
The socket can be created on the host with `nc -U -l /Users/virtuser/vfkit-5.sock`,
and the guest can connect to it with `nc --vsock 2 5`.


This allows virtio-vsock communication from the host to the guest over vsock port 6:
```
--device virtio-vsock,port=6,socketURL=/Users/virtuser/vfkit-6.sock,connect
```
The socket can be created on the guest with `nc --vsock --listen 3 6`,
and the host can connect to it with `nc -U /Users/virtuser/vfkit-6.sock,connect`.


### File Sharing

#### Description

The `-device virtio-fs` option allows to share directories between the host and the guest. The sharing will be done using virtio-fs.
The share can be mounted in the guest with `mount -t virtiofs vfkitTag /mnt`, with `vfkitTag` corresponding to the value of the `mountTag` option.


#### Arguments
- `sharedDir`: absolute path to the host directory to share with the guest.
- `mountTag`: tag which will be used to mount the shared directory in the guest.

#### Example

This will share `/Users/virtuser/vfkit` with the guest:
```
--device virtio-fs,sharedDir=/Users/virtuser/vfkit/,mountTag=vfkit-share
```

The share can then be mounted in Linux guests with:
```
mount -t virtiofs vfkit-share /mount
```

and on macOS with:
```
mkdir /tmp/tag && mount_virtiofs vfkit-share /tmp/tag
```

### Rosetta

#### Description

The `-device rosetta` option allows to use Rosetta to run x86_64 binaries in an arm64 linux VM. This option will share a directory containing the rosetta binaries over virtio-fs.
The share can be mounted in the guest with `mount -t virtiofs vfkitTag /mnt`, with `vfkitTag` corresponding to the value of the `mountTag` option.
Then, [`binfmt`](https://docs.kernel.org/admin-guide/binfmt-misc.html) needs to be configured to use this rosetta binary for x86_64 executables.
On systems using systemd, this can be achieved by creating a /etc/binfmt.d/rosetta.conf file with this content (`/mnt/rosetta` is the full path to the rosetta binary):
```
:rosetta:M::\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x3e\x00:\xff\xff\xff\xff\xff\xfe\xfe\x00\xff\xff\xff\xff\xff\xff\xff\xff\xfe\xff\xff\xff:/mnt/rosetta:F
```
and then running `systemctl restart systemd-binfmt`.

This option is only available on machine with Apple CPUs, `vfkit` will fail with an error if it's used on Intel machines.

See https://developer.apple.com/documentation/virtualization/running_intel_binaries_in_linux_vms_with_rosetta?language=objc for more details.


#### Arguments
- `mountTag`: tag which will be used to mount the rosetta share in the guest.
- `install`: indicates to automatically install rosetta on systems where it's missing. By default, an error will be reported if `--device rosetta` is used when rosetta is not installed.
- `ignoreIfMissing`: indicates if vfkit has to keep executing even though rosetta is not installed. It may happen that the user cancels rosetta installation or the installation fails for other reasons. By default, if rosetta installation fails and vfkit cannot find it, vfkit exits with an error.

#### Example

This adds rosetta support to the guest:
```
--device rosetta,mountTag=rosetta-share
```

The share can then be mounted with `mount -t virtiofs rosetta-share /mnt`.


### GPU

#### Description

The `--device virtio-gpu` option allows the user to add graphical devices to the virtual machine.

#### Arguments
- `width`: the horizontal resolution of the graphical device's resolution. Defaults to 800
- `height`: the vertical resolution of the graphical device's resolution. Defaults to 600

#### Example

`--device virtio-gpu,width=1920,height=1080`


### Input

#### Description

The `--device virtio-input` option allows the user to add an input device to the virtual machine. This currently supports `pointing` and `keyboard` devices.

#### Arguments

None

#### Example

`--device virtio-input,pointing`


## RESTful API

To interact with the RESTful API, append a valid scheme to your base command: `--restful-uri tcp://localhost:8081`.

### Get the virtual machine's state

Obtain the state of the virtual machine that is being run by vfkit.

Request:
```HTTP
GET /vm/state
```

Response:
`{ "state": string, "canStart": bool, "canPause": bool, "canResume": bool, "canStop": bool, "canHardStop": bool }`

`canHardStop` is only supported on macOS 12 and newer, false will always be returned on older versions.
`state` is one of `VirtualMachineStateRunning`, `VirtualMachineStateStopped`, `VirtualMachineStatePaused`, `VirtualMachineStateError`, `VirtualMachineStateStarting`, `VirtualMachineStatePausing`, `VirtualMachineStateResuming`, `VirtualMachineStateStopping`, `VirtualMachineStateSaving`, or `VirtualMachineStateRestoring`.

### Change the virtual machine's state

Change the state of the virtual machine. Valid state values are:
* HardStop
* Pause
* Resume
* Stop

```HTTP
POST /vm/state { "state": "new value"}
```
Response: `HTTP 200`

### Inspect VM

Get description of the virtual machine

```HTTP
GET /vm/inspect
```

Response: `{ "cpus": uint, "memory": uint64, "devices": []config.VirtIODevice }`

## Enabling a Graphical User Interface

### Add a virtio-gpu device

In order to successfully start a graphical application window, a virtio-gpu device must be added to the virtual machine.

### Pass the `--gui` flag

In order to tell vfkit that you want to start a graphical application window, you need to pass the `--gui` flag in your command.

### Usage

Proper use of this flag may look similar to the following section of a command:
```bash
--device virtio-input,keyboard --device virtio-input,pointing --device virtio-gpu,width=1920,height=1080 --gui
```

### Ignition

#### Description

The `--ignition` option allows you to specify a configuration file for Ignition. Vfkit will open a vsock connection between the host and the guest and start a lightweight HTTP server to push the configuration file to Ignition.

You can find example configurations and more details about Ignition at https://coreos.github.io/ignition/ 

#### Example

This command provisions the configuration file to Ignition on the guest
```
--ignition configuration-path
```
