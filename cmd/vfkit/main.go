//go:build darwin
// +build darwin

/*
Copyright 2021, Red Hat, Inc - All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/cmdline"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/crc-org/vfkit/pkg/rest"
	restvf "github.com/crc-org/vfkit/pkg/rest/vf"
	"github.com/crc-org/vfkit/pkg/vf"
	"github.com/docker/go-units"
	log "github.com/sirupsen/logrus"
)

func newLegacyBootloader(opts *cmdline.Options) config.Bootloader {
	if opts.VmlinuzPath == "" && opts.KernelCmdline == "" && opts.InitrdPath == "" {
		return nil
	}

	return config.NewLinuxBootloader(
		opts.VmlinuzPath,
		opts.KernelCmdline,
		opts.InitrdPath,
	)
}

func newBootloaderConfiguration(opts *cmdline.Options) (config.Bootloader, error) {
	legacyBootloader := newLegacyBootloader(opts)

	if legacyBootloader != nil {
		return legacyBootloader, nil
	}

	return config.BootloaderFromCmdLine(opts.Bootloader.GetSlice())
}

func newVMConfiguration(opts *cmdline.Options) (*config.VirtualMachine, error) {
	bootloader, err := newBootloaderConfiguration(opts)
	if err != nil {
		return nil, err
	}

	log.Info(opts)
	log.Infof("boot parameters: %+v", bootloader)
	log.Info()

	vmConfig := config.NewVirtualMachine(
		opts.Vcpus,
		uint64(opts.MemoryMiB*units.MiB),
		bootloader,
	)
	log.Info("virtual machine parameters:")
	log.Infof("\tvCPUs: %d", opts.Vcpus)
	log.Infof("\tmemory: %d MiB", opts.MemoryMiB)
	log.Info()

	if err := vmConfig.AddTimeSyncFromCmdLine(opts.TimeSync); err != nil {
		return nil, err
	}

	if err := vmConfig.AddDevicesFromCmdLine(opts.Devices); err != nil {
		return nil, err
	}

	return vmConfig, nil
}

var errVMStateTimeout = fmt.Errorf("timeout waiting for VM state")

func waitForVMState(vm *vz.VirtualMachine, state vz.VirtualMachineState) error {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGPIPE)

	for {
		select {
		case s := <-signalCh:
			log.Debugf("ignoring signal %v", s)
		case newState := <-vm.StateChangedNotify():
			if newState == state {
				return nil
			}
			if newState == vz.VirtualMachineStateError {
				return fmt.Errorf("hypervisor virtualization error")
			}
		case <-time.After(5 * time.Second):
			return errVMStateTimeout
		}
	}
}

func runVFKit(vmConfig *config.VirtualMachine, opts *cmdline.Options) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	vzVMConfig, err := vf.ToVzVirtualMachineConfig(vmConfig)
	if err != nil {
		return err
	}

	gpuDevs := vmConfig.VirtioGPUDevices()
	if opts.UseGUI && len(gpuDevs) > 0 {
		gpuDevs[0].UsesGUI = true
	}

	vm, err := vz.NewVirtualMachine(vzVMConfig)
	if err != nil {
		return err
	}

	// Do not enable the rests server if user sets scheme to None
	if opts.RestfulURI != cmdline.DefaultRestfulURI {
		restVM := restvf.NewVzVirtualMachine(vm, vzVMConfig)
		srv, err := rest.NewServer(restVM, restVM, opts.RestfulURI)
		if err != nil {
			return err
		}
		srv.Start()
	}
	return runVirtualMachine(vmConfig, vm)
}

func runVirtualMachine(vmConfig *config.VirtualMachine, vm *vz.VirtualMachine) error {
	err := vm.Start()
	if err != nil {
		return err
	}

	err = waitForVMState(vm, vz.VirtualMachineStateRunning)
	if err != nil {
		return err
	}
	log.Infof("virtual machine is running")

	for _, vsock := range vmConfig.VirtioVsockDevices() {
		port := vsock.Port
		socketURL := vsock.SocketURL
		if socketURL == "" {
			// the timesync code adds a vsock device without an associated URL.
			continue
		}
		var listenStr string
		if vsock.Listen {
			listenStr = " (listening)"
		}
		log.Infof("Exposing vsock port %d on %s%s", port, socketURL, listenStr)
		if err := vf.ExposeVsock(vm, port, socketURL, vsock.Listen); err != nil {
			log.Warnf("error exposing vsock port %d: %v", port, err)
		}
	}

	if err := setupGuestTimeSync(vm, vmConfig.TimeSync()); err != nil {
		log.Warnf("Error configuring guest time synchronization")
		log.Debugf("%v", err)
	}

	log.Infof("waiting for VM to stop")

	errCh := make(chan error, 1)
	go func() {
		for {
			err := waitForVMState(vm, vz.VirtualMachineStateStopped)
			if err == nil {
				log.Infof("VM is stopped")
				errCh <- nil
				return
			}
			if !errors.Is(err, errVMStateTimeout) {
				errCh <- fmt.Errorf("virtualization error: %v", err)
				return
			}
			// errVMStateTimeout -> keep looping
		}
	}()

	for _, gpuDev := range vmConfig.VirtioGPUDevices() {
		if gpuDev.UsesGUI {
			runtime.LockOSThread()
			err := vm.StartGraphicApplication(float64(gpuDev.Height), float64(gpuDev.Width))
			runtime.UnlockOSThread()
			if err != nil {
				return err
			}
			break
		}
	}

	return <-errCh
}
