//go:build darwin

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
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/cmdline"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/crc-org/vfkit/pkg/process"
	"github.com/crc-org/vfkit/pkg/rest"
	restvf "github.com/crc-org/vfkit/pkg/rest/vf"
	"github.com/crc-org/vfkit/pkg/vf"
	"github.com/kdomanski/iso9660"
	log "github.com/sirupsen/logrus"

	"github.com/crc-org/vfkit/pkg/util"
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

	log.Debugf("parsed options: %+v", opts)
	log.Debugf("boot parameters: %+v", bootloader)

	vmConfig := config.NewVirtualMachine(
		opts.Vcpus,
		uint64(opts.MemoryMiB),
		bootloader,
	)
	if opts.Nested && !vz.IsNestedVirtualizationSupported() {
		return nil, fmt.Errorf("nested virtualization is not supported")
	}
	vmConfig.Nested = opts.Nested
	log.Info("virtual machine parameters:")
	log.Infof("\tvCPUs: %d", opts.Vcpus)
	log.Infof("\tmemory: %d MiB", opts.MemoryMiB)
	log.Info()

	if err := vmConfig.AddTimeSyncFromCmdLine(opts.TimeSync); err != nil {
		return nil, err
	}

	cloudInitISO, err := generateCloudInitImage(opts.CloudInitFiles.GetSlice())
	if err != nil {
		return nil, err
	}

	// if it generated a valid cloudinit config ISO file we add it to the devices
	if cloudInitISO != "" {
		opts.Devices = append(opts.Devices, fmt.Sprintf("virtio-blk,path=%s", cloudInitISO))
	}

	if opts.PidFile != "" {
		execPath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("could not determine executable path: %w", err)
		}
		vfProcess := process.New(os.Args[0], opts.PidFile, execPath)
		pid := os.Getpid()
		err = vfProcess.WritePidFile(pid)
		if err != nil {
			return nil, fmt.Errorf("could not write PID: %w", err)
		}
	}

	if err := vmConfig.AddDevicesFromCmdLine(opts.Devices); err != nil {
		return nil, err
	}

	if opts.UseGUI {
		if len(vmConfig.VirtioGPUDevices()) == 0 {
			log.Warnf("--gui flag specified but no virtio-gpu device configured, automatically adding it")
			dev, err := config.VirtioGPUNew()
			if err != nil {
				return nil, fmt.Errorf("failed to add virtio-gpu device: %w", err)
			}
			dev.(*config.VirtioGPU).UsesGUI = true
			err = vmConfig.AddDevice(dev)
			if err != nil {
				return nil, fmt.Errorf("failed to add virtio-gpu device: %w", err)
			}
		}
		if len(vmConfig.VirtioInputDevices()) == 0 {
			log.Warnf("--gui flag specified but no virtio-input device configured, automatically adding it")
			dev, err := config.VirtioInputNew(config.VirtioInputKeyboardDevice)
			if err != nil {
				return nil, fmt.Errorf("failed to add virtio-input device: %w", err)
			}
			err = vmConfig.AddDevice(dev)
			if err != nil {
				return nil, fmt.Errorf("failed to add virtio-input device: %w", err)
			}
		}
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

	shutdownFunc := func() {
		log.Debugf("shutting down...")
		stopped, err := vfVM.RequestStop()
		if err != nil {
			log.Errorf("failed to shutdown VM: %v", err)
		} else if !stopped {
			log.Warnf("VM did not acknowledge stop request")
		}
		if err := waitForVMState(vfVM, vz.VirtualMachineStateStopped, time.After(5*time.Second)); err != nil {
			log.Warnf("failed to wait for VM stop: %v, forcing stop", err)
			if forceErr := vfVM.Stop(); forceErr != nil {
				log.Errorf("failed to force stop VM: %v", forceErr)
			}
		} else {
			log.Debugf("VM stopped gracefully")
		}

	}
	util.SetupExitSignalHandling(shutdownFunc)
	return runVirtualMachine(vmConfig, vfVM)
}

func runVirtualMachine(vmConfig *config.VirtualMachine, vm *vf.VirtualMachine) error {
	if vm.Config().Ignition != nil {
		go func() {
			if err := startIgnitionProvisionerServer(vm, vmConfig.Ignition.ConfigPath, vmConfig.Ignition.VsockPort); err != nil {
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
			// timesync and ignition add a vsock device without an associated URL.
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

	if err := vf.ListenNetworkBlockDevices(vm); err != nil {
		log.Debugf("%v", err)
		return err
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

func startIgnitionProvisionerServer(vm *vf.VirtualMachine, configPath string, vsockPort uint32) error {
	ignitionReader, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer ignitionReader.Close()

	vsockDevices := vm.SocketDevices()
	if len(vsockDevices) != 1 {
		return fmt.Errorf("VM has too many/not enough virtio-vsock devices (%d)", len(vsockDevices))
	}
	listener, err := vsockDevices[0].Listen(vsockPort)
	if err != nil {
		return err
	}

	defer func() {
		if err := listener.Close(); err != nil {
			log.Error(err)
		}
	}()

	return startIgnitionProvisionerServerInternal(ignitionReader, listener)
}

func startIgnitionProvisionerServerInternal(ignitionReader io.ReadSeeker, listener net.Listener) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		http.ServeContent(w, req, "", time.Time{}, ignitionReader)
	})

	srv := &http.Server{
		Handler:           mux,
		Addr:              listener.Addr().String(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Debugf("ignition socket: %s", listener.Addr().String())
	return srv.Serve(listener)
}

// it generates a cloud init image by taking the files passed by the user
// as cloud-init expects files with a specific name (e.g user-data, meta-data) we check the filenames to retrieve the correct info
// if some file is not passed by the user, an empty file will be copied to the cloud-init ISO to
// guarantee it to work (user-data and meta-data files are mandatory and both must exists, even if they are empty)
// if both files are missing it returns an error
func generateCloudInitImage(files []string) (string, error) {
	if len(files) == 0 {
		return "", nil
	}

	configFiles := map[string]io.Reader{
		"user-data":      nil,
		"meta-data":      nil,
		"network-config": nil,
	}

	hasConfigFile := false
	for _, path := range files {
		if path == "" {
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer file.Close()

		filename := filepath.Base(path)
		if _, ok := configFiles[filename]; ok {
			if filename == "user-data" || filename == "meta-data" {
				hasConfigFile = true
			}
			configFiles[filename] = file
		}
	}

	if !hasConfigFile {
		return "", fmt.Errorf("cloud-init needs user-data and meta-data files to work")
	}

	return createCloudInitISO(configFiles)
}

// It generates a temp ISO file containing the files passed by the user
// It also register an exit handler to delete the file when vfkit exits
func createCloudInitISO(files map[string]io.Reader) (string, error) {
	writer, err := iso9660.NewWriter()
	if err != nil {
		return "", fmt.Errorf("failed to create writer: %w", err)
	}

	defer func() {
		if err := writer.Cleanup(); err != nil {
			log.Error(err)
		}
	}()

	for name, reader := range files {
		// if reader is nil, we set it to an empty file
		if reader == nil {
			reader = bytes.NewReader([]byte{})
		}
		err = writer.AddFile(reader, name)
		if err != nil {
			return "", fmt.Errorf("failed to add %s file: %w", name, err)
		}
	}

	isoFile, err := os.CreateTemp("", "vfkit-cloudinit-")
	if err != nil {
		return "", fmt.Errorf("unable to create temporary cloud-init ISO file: %w", err)
	}

	defer func() {
		if err := isoFile.Close(); err != nil {
			log.Error(fmt.Errorf("failed to close cloud-init ISO file: %w", err))
		}
	}()

	// register handler to remove isoFile when exiting
	util.RegisterExitHandler(func() {
		os.Remove(isoFile.Name())
	})

	err = writer.WriteTo(isoFile, "cidata")
	if err != nil {
		return "", fmt.Errorf("failed to write cloud-init ISO image: %w", err)
	}

	return isoFile.Name(), nil
}
