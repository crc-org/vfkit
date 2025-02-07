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
	"fmt"
	"io"
	"net"
	"net/http"
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
		uint64(opts.MemoryMiB),
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

	if err := vmConfig.AddIgnitionFileFromCmdLine(opts.IgnitionPath); err != nil {
		return nil, fmt.Errorf("failed to add ignition file: %w", err)
	}

	return vmConfig, nil
}

func waitForVMState(vm *vf.VirtualMachine, state vz.VirtualMachineState, timeout <-chan time.Time) error {
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
		case <-timeout:
			return fmt.Errorf("timeout waiting for VM state")
		}
	}
}

func runVFKit(vmConfig *config.VirtualMachine, opts *cmdline.Options) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	gpuDevs := vmConfig.VirtioGPUDevices()
	if opts.UseGUI && len(gpuDevs) > 0 {
		gpuDevs[0].UsesGUI = true
	}

	vfVM, err := vf.NewVirtualMachine(*vmConfig)
	if err != nil {
		return err
	}

	// Do not enable the rests server if user sets scheme to None
	if opts.RestfulURI != cmdline.DefaultRestfulURI {
		restVM := restvf.NewVzVirtualMachine(vfVM)
		srv, err := rest.NewServer(restVM, restVM, opts.RestfulURI)
		if err != nil {
			return err
		}
		srv.Start()
	}
	return runVirtualMachine(vmConfig, vfVM)
}

func runVirtualMachine(vmConfig *config.VirtualMachine, vm *vf.VirtualMachine) error {
	if vm.Config().Ignition != nil {
		go func() {
			file, err := os.Open(vmConfig.Ignition.ConfigPath)
			if err != nil {
				log.Error(err)
			}
			defer file.Close()
			reader := file
			if err := startIgnitionProvisionerServer(reader, vmConfig.Ignition.SocketPath); err != nil {
				log.Error(err)
			}
			log.Debug("ignition vsock server exited")
		}()
	}

	if err := vm.Start(); err != nil {
		return err
	}

	if err := waitForVMState(vm, vz.VirtualMachineStateRunning, time.After(5*time.Second)); err != nil {
		return err
	}
	log.Infof("virtual machine is running")

	vsockDevs := vmConfig.VirtioVsockDevices()
	for _, vsock := range vsockDevs {
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
		closer, err := vf.ExposeVsock(vm, port, socketURL, vsock.Listen)
		if err != nil {
			log.Warnf("error exposing vsock port %d: %v", port, err)
			continue
		}
		defer closer.Close()
	}

	if err := setupGuestTimeSync(vm, vmConfig.TimeSync()); err != nil {
		log.Warnf("Error configuring guest time synchronization")
		log.Debugf("%v", err)
	}

	log.Infof("waiting for VM to stop")

	errCh := make(chan error, 1)
	go func() {
		if err := waitForVMState(vm, vz.VirtualMachineStateStopped, nil); err != nil {
			errCh <- fmt.Errorf("virtualization error: %v", err)
		} else {
			log.Infof("VM is stopped")
			errCh <- nil
		}
	}()

	for _, gpuDev := range vmConfig.VirtioGPUDevices() {
		if gpuDev.UsesGUI {
			runtime.LockOSThread()
			err := vm.StartGraphicApplication(float64(gpuDev.Width), float64(gpuDev.Height))
			runtime.UnlockOSThread()
			if err != nil {
				return err
			}
			break
		}
	}

	return <-errCh
}

func startIgnitionProvisionerServer(ignitionReader io.Reader, ignitionSocketPath string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.Copy(w, ignitionReader)
		if err != nil {
			log.Errorf("failed to serve ignition file: %v", err)
		}
	})

	listener, err := net.Listen("unix", ignitionSocketPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := listener.Close(); err != nil {
			log.Error(err)
		}
	}()

	srv := &http.Server{
		Handler:           mux,
		Addr:              ignitionSocketPath,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Debugf("ignition socket: %s", ignitionSocketPath)
	return srv.Serve(listener)
}
