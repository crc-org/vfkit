vfkit - Simple command line tool to start VMs through virtualization framework
====

### Introduction

vfkit offers a command-line interface to start virtual machines using virtualization framework.
It also provides a github.com/code-ready/vfkit/tree/main/pkg/client go package.
This package provides a native go API to generate the vfkit command line.


### Installation

vfkit is available in the `crc` brew tap:

```
brew tap cfergeau/crc
brew install vfkit
```


### Usage

See https://github.com/crc-org/vfkit/blob/main/doc/usage.md


### Presentations

`vfkit` has been presented at a few conferences:
- [Containers Plumbing 2023](https://crc.dev/blog/posts/2023-03-22-containers-plumbing/)
- [FOSDEM 2023](https://fosdem.org/2023/schedule/event/govfkit/)


### Background

The work in this repository makes use of https://github.com/Code-Hex/vz which provides go bindings for macOS virtualization framework.
The lifetime of virtual machines created using the virtualization framework is tied to the filetime of the process where they were created.
When using `Code-Hex/vz`, this means the virtual machine will be terminated at the end of the go process using these bindings.
Spawning a `vfkit` process gives more flexibility and more control over the lifetime of the virtual machine.


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
