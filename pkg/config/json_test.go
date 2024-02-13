package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type jsonTest struct {
	newVM        func(*testing.T) *VirtualMachine
	expectedJSON string
}

var jsonTests = map[string]jsonTest{
	"TestLinuxVM": {
		newVM:        newLinuxVM,
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","kernelCmdLine":"/initrd","initrdPath":"console=hvc0"}}`,
	},
	"TestUEFIVM": {
		newVM:        newUEFIVM,
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"efiBootloader","efiVariableStorePath":"/variable-store","createVariableStore":false}}`,
	},
	"TestTimeSync": {
		newVM: func(t *testing.T) *VirtualMachine {
			vm := newLinuxVM(t)
			timesync, err := TimeSyncNew(1234)
			require.NoError(t, err)
			vm.Timesync = timesync.(*TimeSync)
			return vm
		},
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","kernelCmdLine":"/initrd","initrdPath":"console=hvc0"},"timesync":{"vsockPort":1234}}`,
	},
	"TestVirtioRNG": {
		newVM: func(t *testing.T) *VirtualMachine {
			vm := newLinuxVM(t)
			virtioRng, err := VirtioRngNew()
			require.NoError(t, err)
			err = vm.AddDevice(virtioRng)
			require.NoError(t, err)
			return vm
		},
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","kernelCmdLine":"/initrd","initrdPath":"console=hvc0"},"devices":[{"kind":"virtiorng"}]}`,
	},
	"TestMultipleVirtioBlk": {
		newVM: func(t *testing.T) *VirtualMachine {
			vm := newLinuxVM(t)
			virtioBlk, err := VirtioBlkNew("/virtioblk1")
			require.NoError(t, err)
			err = vm.AddDevice(virtioBlk)
			require.NoError(t, err)
			virtioBlk, err = VirtioBlkNew("/virtioblk2")
			require.NoError(t, err)
			virtioBlk.SetDeviceIdentifier("virtio-blk2")
			err = vm.AddDevice(virtioBlk)
			require.NoError(t, err)
			return vm
		},
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","kernelCmdLine":"/initrd","initrdPath":"console=hvc0"},"devices":[{"kind":"virtioblk","devName":"virtio-blk","imagePath":"/virtioblk1"},{"kind":"virtioblk","devName":"virtio-blk","imagePath":"/virtioblk2","deviceIdentifier":"virtio-blk2"}]}`,
	},
	"TestAllVirtioDevices": {
		newVM: func(t *testing.T) *VirtualMachine {
			vm := newLinuxVM(t)
			// virtio-serial
			dev, err := VirtioSerialNew("/virtioserial")
			require.NoError(t, err)
			err = vm.AddDevice(dev)
			require.NoError(t, err)
			// virtio-input
			dev, err = VirtioInputNew(VirtioInputKeyboardDevice)
			require.NoError(t, err)
			err = vm.AddDevice(dev)
			require.NoError(t, err)
			// virtio-gpu
			dev, err = VirtioGPUNew()
			require.NoError(t, err)
			err = vm.AddDevice(dev)
			require.NoError(t, err)
			// virtio-net
			dev, err = VirtioNetNew("00:11:22:33:44:55")
			require.NoError(t, err)
			err = vm.AddDevice(dev)
			require.NoError(t, err)
			// virtio-rng
			dev, err = VirtioRngNew()
			require.NoError(t, err)
			err = vm.AddDevice(dev)
			require.NoError(t, err)
			// virtio-blk
			dev, err = VirtioBlkNew("/virtioblk")
			require.NoError(t, err)
			err = vm.AddDevice(dev)
			require.NoError(t, err)
			// virtio-vsock
			dev, err = VirtioVsockNew(1234, "/virtiovsock", false)
			require.NoError(t, err)
			err = vm.AddDevice(dev)
			require.NoError(t, err)
			// virtio-fs
			fs, err := VirtioFsNew("/virtiofs", "tag")
			require.NoError(t, err)
			// USB mass storage
			usb, err := USBMassStorageNew("/usbmassstorage")
			require.NoError(t, err)
			// rosetta
			rosetta, err := RosettaShareNew("vz-rosetta")
			require.NoError(t, err)
			err = vm.AddDevices(fs, usb, rosetta)
			require.NoError(t, err)

			return vm
		},
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","kernelCmdLine":"/initrd","initrdPath":"console=hvc0"},"devices":[{"kind":"virtioserial","logFile":"/virtioserial"},{"kind":"virtioinput","inputType":"keyboard"},{"kind":"virtiogpu","usesGUI":false,"width":800,"height":600},{"kind":"virtionet","nat":true,"macAddress":"ABEiM0RV"},{"kind":"virtiorng"},{"kind":"virtioblk","devName":"virtio-blk","imagePath":"/virtioblk"},{"kind":"virtiosock","port":1234,"socketURL":"/virtiovsock"},{"kind":"virtiofs","mountTag":"tag","sharedDir":"/virtiofs"},{"kind":"usbmassstorage","devName":"usb-mass-storage","imagePath":"/usbmassstorage"},{"kind":"rosetta","mountTag":"vz-rosetta","installRosetta":false}]}`,
	},
}

type invalidJSONTest struct {
	json string
}

var invalidJSONTests = map[string]invalidJSONTest{
	"TestEmptyBootloaderKind": {
		json: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"empty",vmlinuzPath":"/vmlinuz","kernelCmdLine":"/initrd","initrdPath":"console=hvc0"}}`,
	},
	"TestInvalidBootloaderKind": {
		json: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"invalid",vmlinuzPath":"/vmlinuz","kernelCmdLine":"/initrd","initrdPath":"console=hvc0"}}`,
	},
	"TestMissingBootloaderKind": {
		json: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"vmlinuzPath":"/vmlinuz","kernelCmdLine":"/initrd","initrdPath":"console=hvc0"}}`,
	},
	"TestEmptyDeviceKind": {
		json: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","kernelCmdLine":"/initrd","initrdPath":"console=hvc0"},"devices":[{"kind":"","devName":"virtio-blk","imagePath":"/virtioblk1"}]}`,
	},
	"TestInvalidDeviceKind": {
		json: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","kernelCmdLine":"/initrd","initrdPath":"console=hvc0"},"devices":[{"kind":"invalid","devName":"virtio-blk","imagePath":"/virtioblk1"}]}`,
	},
	"TestMissingDeviceKind": {
		json: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","kernelCmdLine":"/initrd","initrdPath":"console=hvc0"},"devices":[{"devName":"virtio-blk","imagePath":"/virtioblk1"}]}`,
	},
}

func TestJSON(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		for name := range jsonTests {
			t.Run(name, func(t *testing.T) {
				test := jsonTests[name]
				testJSON(t, &test)
			})
		}
		for name := range invalidJSONTests {
			t.Run(name, func(t *testing.T) {
				test := invalidJSONTests[name]
				testInvalidJSON(t, &test)
			})
		}
	})
}

func testJSON(t *testing.T, test *jsonTest) {
	vm := test.newVM(t)
	data, err := json.Marshal(vm)
	require.NoError(t, err)
	require.JSONEq(t, test.expectedJSON, string(data))

	var unmarshalledVM VirtualMachine
	err = json.Unmarshal([]byte(test.expectedJSON), &unmarshalledVM)
	require.NoError(t, err)

	require.Equal(t, *vm, unmarshalledVM)
}

func testInvalidJSON(t *testing.T, test *invalidJSONTest) {
	var vm VirtualMachine
	err := json.Unmarshal([]byte(test.json), &vm)
	require.Error(t, err)
}

func newLinuxVM(*testing.T) *VirtualMachine {
	bootloader := NewLinuxBootloader("/vmlinuz", "/initrd", "console=hvc0")
	vm := NewVirtualMachine(3, 4_000, bootloader)

	return vm
}

func newUEFIVM(_ *testing.T) *VirtualMachine {
	bootloader := NewEFIBootloader("/variable-store", false)
	vm := NewVirtualMachine(3, 4_000, bootloader)

	return vm
}
