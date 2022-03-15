vfkit - Simple command line tool to start VMs through virtualization framework
====

vfkit offers a command-line interface to start virtual machines using virtualization framework

The work in this repository makes use of https://github.com/Code-Hex/vz to create a Linux virtual machine with virtualization.framework using go.

The kernel must be uncompressed before use as no bootloader is used, as
documented in https://www.kernel.org/doc/Documentation/arm64/booting.txt

```
3. Decompress the kernel image
------------------------------

Requirement: OPTIONAL

The AArch64 kernel does not currently provide a decompressor and therefore
requires decompression (gzip etc.) to be performed by the boot loader if a
compressed Image target (e.g. Image.gz) is used.  For bootloaders that do not
implement this requirement, the uncompressed Image target is available instead.
```
