# Using `linux-next` and `dracut` to Build an Uncompressed Kernel and Initrd

> This guide was tested on a **Fedora Cloud VM** running in `vfkit`.

This guide walks you through building an uncompressed kernel (`Image`) from `linux-next` and generating an initial RAM disk (initrd) using `dracut` on Fedora Cloud. This is especially useful for direct kernel boot on `vfkit` via `--bootloader linux` on ARM64 (Apple Silicon, etc.).

## What is `linux-next`?

`linux-next` is a staging branch for patches headed to the next mainline Linux release. Maintainers integrate and test changes here to detect conflicts early.

> See [linux-next docs](https://www.kernel.org/doc/man-pages/linux-next.html) for more information.

## Prerequisites

- You have vfkit installed and working on your system.
- A Fedora Cloud image is available locally.
- Install required packages:

```bash
sudo dnf install -y git make gcc flex bison bc openssl-devel dracut
```

## Step 1: Clone and Build `linux-next`

```bash
git clone https://git.kernel.org/pub/scm/linux/kernel/git/next/linux-next.git
cd linux-next
make defconfig
make -j$(nproc)
```

## Step 2: Locate the Uncompressed Kernel

The build produces:

- `arch/arm64/boot/Image` — the uncompressed kernel for ARM64

For `vfkit` and Apple Silicon, **use the uncompressed `Image`**.

## Step 3: Build Kernel Modules and Initrd with Dracut

### 3.1 Install Modules

```bash
make modules_install INSTALL_MOD_PATH=./modinstall
sudo cp -r modinstall/lib/modules/$(make kernelrelease) /lib/modules/
```

### 3.2 Generate the Initrd

```bash
mkdir -p /tmp/initrd-out
sudo dracut -f /tmp/initrd-out/initrd.img $(make kernelrelease) --kver $(make kernelrelease)
```

## Step 4: Boot with Kernel + Initrd

Once the Image and initrd.img files are built, copy them to your macOS host. Then, run:

```bash
vfkit --cpus 2 --memory 2048 \
--bootloader linux,kernel=/path/to/kernel/Image,initrd=/path/to/kernel/initrd.img,cmdline="root=/dev/vda1 console=hvc0" \
--device virtio-blk,path=/path/to/Fedora-Cloud.raw  \
--device virtio-serial,logFilePath=/tmp/vfkit.log
```

Adjust flags and devices based on your use case.
