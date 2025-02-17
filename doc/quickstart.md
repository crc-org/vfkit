# vfkit quick start

## Introduction

vfkit is a macOS command-line-based hypervisor, which uses [Apple's Virtualization Framework](https://developer.apple.com/documentation/virtualization?language=objc) to run virtual machines.
You start a virtual machine by running vfkit with a set of arguments describing the virtual machine configuration/hardware.
When vfkit stops, the virtual machine stops running.
It requires macOS 12 or newer, and runs on both Intel and Apple silicon Macs.
It may build and run on macOS 11, but this platform is no longer tested as it's [out of support](https://endoflife.date/macos).
File sharing is only available on macOS 12 or newer.
UEFI boot and graphical user interface support are only available on macOS 13 or newer.


## Installation

You can either download vfkit from [its release page](https://github.com/crc-org/vfkit/releases), or install it from [brew](https://brew.sh/):
```
brew install vfkit
```


## Quick start

### Getting a disk image

Your virtual machine will need an operating system to run, so you need to download a disk image first.
The image needs to be in the raw or iso format. Please note that qcow2 or
VirtualBox images cannot be used by vfkit.

For example, Fedora images can be downloaded with:
```
# For Apple silicon Macs
curl -L -O https://download.fedoraproject.org/pub/fedora/linux/releases/38/Cloud/aarch64/images/Fedora-Cloud-Base-38-1.6.aarch64.raw.xz
xz -d ./Fedora-Cloud-Base-38-1.6.aarch64.raw.xz
mv Fedora-Cloud-Base-38-1.6.aarch64.raw vfkit-test-image.raw

# For Intel Macs
curl -L https://download.fedoraproject.org/pub/fedora/linux/releases/38/Cloud/x86_64/images/Fedora-Cloud-Base-38-1.6.x86_64.raw.xz
xz -d ./Fedora-Cloud-Base-38-1.6.x86_64.raw.xz
mv Fedora-Cloud-Base-38-1.6.x86_64.raw vfkit-test-image.raw
```


### Starting a virtual machine


Now that we have a disk image, we can start a virtual machine with 2 virtual CPUs and 2GiB of RAM:

```
vfkit \
    --cpus 2 --memory 2048 \
    --bootloader efi,variable-store=efi-variable-store,create \
    --device virtio-blk,path=vfkit-test-image.raw
```

No logs from the virtual machine are displayed in the terminal where vfkit was started, but it should show:
```
INFO[0000] virtual machine is running
INFO[0000] waiting for VM to stop
```

The virtual machine will be running until you hit Ctrl+C or kill the vfkit process.
If you are using an image or an older macOS version which does not support UEFI boot, you can use the `linux` bootloader.
This requires a separate kernel, initrd file and kernel command-line arguments.
Details can be found in the [usage instructions](https://github.com/crc-org/vfkit/blob/main/doc/usage.md#linux-bootloader).

### Adding a GUI

To run a VM with a graphical user interface, append the necessary flags to your vfkit command:
`--device virtio-input,keyboard --device virtio-input,pointing --device virtio-gpu,width=800,height=600 --gui`

### Adding a serial console for boot logs

To get some logs from the virtual machine, we can add a virtio-serial device to vfkit command-line:
```
--device virtio-serial,stdio
```

The logs will be shown in the terminal where vfkit was started.
With the Fedora image, only the login prompt is shown after approximately 30s.
On more verbose images, the boot logs are only shown late in the boot process as they only start to appear after the virtio-serial module and associated console have been loaded and configured.


### Adding a network card

To make the interactions with the virtual machine easier, we can add a virtio-net device to it:
```
--device virtio-net,nat
```

After booting, the Fedora image prints the IP address of the VM in the serial console before the login prompt.

You can specify the mac address of the network interface on the command-line:
```
--device virtio-net,nat,mac=72:20:43:d4:39:63
```

This allows to lookup the IP which was assigned to the virtual machine by searching for the mac address in the `/var/db/dhcpd_leases` file on the host.


### Next steps


Once you have a virtual machine up and running, here are some additional features which can be useful:
- [sparse/copy-on-write disk images](https://github.com/crc-org/vfkit/blob/main/doc/usage.md#thin-images)
- [host/guest communication over virtio-vsock](https://github.com/crc-org/vfkit/blob/main/doc/usage.md#virtio-vsock-communication)
- [host/guest file sharing with virtio-fs](https://github.com/crc-org/vfkit/blob/main/doc/usage.md#file-sharing)
- [Rosetta support to run x86_64 binaries in virtual machines on Apple silicon Macs](https://github.com/crc-org/vfkit/blob/main/doc/usage.md#rosetta)
- [REST API to control the virtual machine](https://github.com/crc-org/vfkit/blob/main/doc/usage.md#restful-service)
- [user-mode networking with the `gvproxy` command from gvisor-tap-vsock](https://github.com/containers/gvisor-tap-vsock)

Full documentation of vfkit's various features is documented in the [usage guide](https://github.com/crc-org/vfkit/blob/main/doc/usage.md).

Any questions/issues/... with vfkit can be reported [on GitHub](https://github.com/crc-org/vfkit/issues/new).
