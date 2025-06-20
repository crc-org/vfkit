# Accessing Boot Messages in vfkit

> **Note:** This guide primarily demonstrates usage with Fedora Cloud and CoreOS images.

There are different ways to access boot logs in vfkit, depending on the following factors:

## Boot Method

### Direct Kernel Boot (`--bootloader linux`)

This method requires the user to provide a kernel image and an initrd. You can pass kernel command-line arguments to configure logging output. In the case of `vfkit`, use `console=hvc0`. To access the logs on the host, attach a serial device with `logFilePath` parameter set to the desired location on the host. See the [vfkit serial port usage documentation](https://github.com/crc-org/vfkit/blob/main/doc/usage.md#serial-port) for more.

```bash
vfkit --cpus 2 --memory 2048 \
  --bootloader linux,kernel=/path/to/Image,initrd=/path/to/initrd.img,cmdline="root=/dev/vda1 console=hvc0" \
  --device virtio-blk,path=/path/to/Fedora-Cloud-Image.raw \
  --device virtio-serial,logFilePath=/tmp/vm-console.log
```

Logs will appear in `/tmp/vm-console.log` on the host.

### EFI Boot (`--bootloader efi`)

This is the default mode for Fedora Cloud and CoreOS images. You can enable logging in two ways:

#### 1. Editing GRUB in the VM (for Fedora Cloud)

```bash
sudo vi /etc/default/grub
```

Add or update the following lines:

```ini
GRUB_CMDLINE_LINUX="console=hvc0"
GRUB_TERMINAL="serial console"
```

Then regenerate the GRUB config and reboot:

```bash
sudo grub2-mkconfig -o /boot/grub2/grub.cfg
```

Launch `vfkit` with:

```bash
vfkit --cpus 2 --memory 2048 \
  --bootloader efi,variable-store=out/efistore.nvram,create \
  --device virtio-blk,path=/path/to/Fedora-Cloud-Image.raw \
  --device virtio-serial,logFilePath=/tmp/vm-console.log
```

#### 2. Using `rpm-ostree` command (for Fedora CoreOS)

You can get boot logs by running the following command on guest:

```bash
sudo rpm-ostree kargs --append console=hvc0 --reboot
```

Then boot with:

```bash
vfkit --cpus 2 --memory 2048 \
  --bootloader efi,variable-store=out/efistore.nvram,create \
  --device virtio-blk,path=/path/to/fedora-coreos.raw \
  --device virtio-net,nat,mac=72:20:43:d4:38:62 \
  --device virtio-serial,logFilePath=/tmp/vfkit.log 
```

You can access the logs in `/tmp/vfkit.log` on the host.

### Custom-Built Images

If you're using a custom kernel and initrd, ensure logging is configured via the kernel command line (e.g., `console=hvc0`). You can then follow the same `--device virtio-serial` method shown above to collect logs.

## Console Output Target

You have multiple ways to view logs from the VM:

1. Use `--device virtio-serial` with parameters like `logFilePath`, `stdin`, and `pty` to redirect VM logs to the host. See the [vfkit serial port usage documentation](https://github.com/crc-org/vfkit/blob/main/doc/usage.md#serial-port) for more.
2. Use `--gui` to open a graphical terminal window for viewing VM logs. See [vfkit GUI usage](https://github.com/crc-org/vfkit/blob/main/doc/usage.md#enabling-a-graphical-user-interface).

> **Note:** The `--gui` option works only with the **EFI bootloader**.

You can use both `--gui` and `--device virtio-serial` simultaneously for flexible access to logs.

## Early Boot Messages

Currently, it's not possible to view early boot messages directly on the host. To inspect them, you can SSH into the VM and run the following command:

```bash
dmesg
```
