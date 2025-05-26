vfkit - Command-line tool to start VMs on macOS
====

### Introduction

vfkit offers a command-line interface to start virtual machines using the [macOS Virtualization framework](https://developer.apple.com/documentation/virtualization).
It also provides a `github.com/crc-org/vfkit/pkg/config` go package.
This package implements a native Go API to generate the vfkit command line.

### Usage

See https://github.com/crc-org/vfkit/blob/main/doc/usage.md


### Presentations

`vfkit` has been presented at a few conferences:
- [Containers Plumbing 2023](https://crc.dev/blog/posts/2023-03-22-containers-plumbing/)
- [FOSDEM 2023](https://fosdem.org/2023/schedule/event/govfkit/)

### Adopters

- [minikube](https://minikube.sigs.k8s.io/) 1.35.0 and newer - minikube quickly sets up a local Kubernetes cluster
- [podman](https://podman.io/) 5.0 and newer - podman is a free software CLI tool to manage containers, pods and images
- [crc](https://crc.dev/) - crc sets up local OpenShift or MicroShift clusters for development and testing purposes
- [ovm](https://github.com/oomol-lab/ovm) - ovm is used by Oomol Studio to run linux containers on macOS

### Installation

vfkit is available in brew:

```
brew install vfkit
```

### Building

From the root direction of this repository, run `make`.
