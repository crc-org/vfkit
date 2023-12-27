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
		expectedJSON: `{"vcpus":3,"memoryBytes":4000000000,"bootloader":{"kind":"linuxBootloader","VmlinuzPath":"/vmlinuz","KernelCmdLine":"/initrd","InitrdPath":"console=hvc0"}}`,
	},
	"TestUEFIVM": {
		newVM:        newUEFIVM,
		expectedJSON: `{"vcpus":3,"memoryBytes":4000000000,"bootloader":{"kind":"efiBootloader","EFIVariableStorePath":"/variable-store","CreateVariableStore":false}}`,
	},
	"TestTimeSync": {
		newVM: func(t *testing.T) *VirtualMachine {
			vm := newLinuxVM(t)
			timesync := TimeSyncNew(1234)
			vm.Timesync = timesync.(*TimeSync)
			return vm
		},
		expectedJSON: `{"vcpus":3,"memoryBytes":4000000000,"bootloader":{"kind":"linuxBootloader","VmlinuzPath":"/vmlinuz","KernelCmdLine":"/initrd","InitrdPath":"console=hvc0"},"timesync":{"VsockPort":1234}}`,
	},
	"TestVirtioRNG": {
		newVM: func(t *testing.T) *VirtualMachine {
			vm := newLinuxVM(t)
			virtioRng := VirtioRngNew()
			vm.AddDevice(virtioRng)
			return vm
		},
		expectedJSON: `{"vcpus":3,"memoryBytes":4000000000,"bootloader":{"kind":"linuxBootloader","VmlinuzPath":"/vmlinuz","KernelCmdLine":"/initrd","InitrdPath":"console=hvc0"},"devices":[{"kind":"virtiorng"}]}`,
	},
	"TestMultipleVirtioBlk": {
		newVM: func(t *testing.T) *VirtualMachine {
			vm := newLinuxVM(t)
			virtioBlk := VirtioBlkNew("/virtioblk1")
			vm.AddDevice(virtioBlk)
			virtioBlk = VirtioBlkNew("/virtioblk2")
			virtioBlk.SetDeviceIdentifier("virtio-blk2")
			vm.AddDevice(virtioBlk)
			return vm
		},
		expectedJSON: `{"vcpus":3,"memoryBytes":4000000000,"bootloader":{"kind":"linuxBootloader","VmlinuzPath":"/vmlinuz","KernelCmdLine":"/initrd","InitrdPath":"console=hvc0"},"devices":[{"kind":"virtioblk","DevName":"virtio-blk","ImagePath":"/virtioblk1","ReadOnly":false,"DeviceIdentifier":""},{"kind":"virtioblk","DevName":"virtio-blk","ImagePath":"/virtioblk2","ReadOnly":false,"DeviceIdentifier":"virtio-blk2"}]}`,
	},
	"TestAllVirtioDevices": {
		newVM: func(t *testing.T) *VirtualMachine {
			vm := newLinuxVM(t)
			// virtio-serial
			dev := VirtioSerialNew("/virtioserial")
			vm.AddDevice(dev)
			// virtio-input
			dev, err := VirtioInputNew(VirtioInputKeyboardDevice)
			require.NoError(t, err)
			vm.AddDevice(dev)
			// virtio-gpu
			dev = VirtioGPUNew()
			require.NoError(t, err)
			vm.AddDevice(dev)
			// virtio-net
			dev, err = VirtioNetNew("00:11:22:33:44:55")
			require.NoError(t, err)
			vm.AddDevice(dev)
			// virtio-rng
			dev = VirtioRngNew()
			vm.AddDevice(dev)
			// virtio-blk
			dev = VirtioBlkNew("/virtioblk")
			vm.AddDevice(dev)
			// virtio-vsock
			dev = VirtioVsockNew(1234, "/virtiovsock", false)
			vm.AddDevice(dev)
			// virtio-fs
			dev = VirtioFsNew("/virtiofs", "tag")
			vm.AddDevice(dev)
			// USB mass storage
			dev = USBMassStorageNew("/usbmassstorage")
			vm.AddDevice(dev)
			// rosetta
			dev = RosettaShareNew("vz-rosetta")
			vm.AddDevice(dev)

			return vm
		},
		expectedJSON: `{"vcpus":3,"memoryBytes":4000000000,"bootloader":{"kind":"linuxBootloader","VmlinuzPath":"/vmlinuz","KernelCmdLine":"/initrd","InitrdPath":"console=hvc0"},"devices":[{"kind":"virtioserial","LogFile":"/virtioserial","UsesStdio":false},{"kind":"virtioinput","inputType":"keyboard"},{"kind":"virtiogpu","usesGUI":false,"width":800,"height":600},{"kind":"virtionet","Nat":true,"MacAddress":"ABEiM0RV","Socket":null,"UnixSocketPath":""},{"kind":"virtiorng"},{"kind":"virtioblk","DevName":"virtio-blk","ImagePath":"/virtioblk","ReadOnly":false,"DeviceIdentifier":""},{"kind":"virtiosock","Port":1234,"SocketURL":"/virtiovsock","Listen":false},{"kind":"virtiofs","MountTag":"tag","SharedDir":"/virtiofs"},{"kind":"usbmassstorage","DevName":"usb-mass-storage","ImagePath":"/usbmassstorage","ReadOnly":false},{"kind":"rosetta","MountTag":"vz-rosetta","InstallRosetta":false}]}`,
	},
}

type invalidJSONTest struct {
	json string
}

var invalidJSONTests = map[string]invalidJSONTest{
	"TestEmptyBootloaderKind": {
		json: `{"vcpus":3,"memoryBytes":4000000000,"bootloader":{"kind":"empty",VmlinuzPath":"/vmlinuz","KernelCmdLine":"/initrd","InitrdPath":"console=hvc0"}}`,
	},
	"TestInvalidBootloaderKind": {
		json: `{"vcpus":3,"memoryBytes":4000000000,"bootloader":{"kind":"invalid",VmlinuzPath":"/vmlinuz","KernelCmdLine":"/initrd","InitrdPath":"console=hvc0"}}`,
	},
	"TestMissingBootloaderKind": {
		json: `{"vcpus":3,"memoryBytes":4000000000,"bootloader":{"VmlinuzPath":"/vmlinuz","KernelCmdLine":"/initrd","InitrdPath":"console=hvc0"}}`,
	},
	"TestEmptyDeviceKind": {
		json: `{"vcpus":3,"memoryBytes":4000000000,"bootloader":{"kind":"linuxBootloader","VmlinuzPath":"/vmlinuz","KernelCmdLine":"/initrd","InitrdPath":"console=hvc0"},"devices":[{"kind":"","DevName":"virtio-blk","ImagePath":"/virtioblk1","ReadOnly":false,"DeviceIdentifier":""}]}`,
	},
	"TestInvalidDeviceKind": {
		json: `{"vcpus":3,"memoryBytes":4000000000,"bootloader":{"kind":"linuxBootloader","VmlinuzPath":"/vmlinuz","KernelCmdLine":"/initrd","InitrdPath":"console=hvc0"},"devices":[{"kind":"invalid","DevName":"virtio-blk","ImagePath":"/virtioblk1","ReadOnly":false,"DeviceIdentifier":""}]}`,
	},
	"TestMissingDeviceKind": {
		json: `{"vcpus":3,"memoryBytes":4000000000,"bootloader":{"kind":"linuxBootloader","VmlinuzPath":"/vmlinuz","KernelCmdLine":"/initrd","InitrdPath":"console=hvc0"},"devices":[{"DevName":"virtio-blk","ImagePath":"/virtioblk1","ReadOnly":false,"DeviceIdentifier":""}]}`,
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
	vm := NewVirtualMachine(3, 4_000_000_000, bootloader)

	return vm
}

func newUEFIVM(_ *testing.T) *VirtualMachine {
	bootloader := NewEFIBootloader("/variable-store", false)
	vm := NewVirtualMachine(3, 4_000_000_000, bootloader)

	return vm
}
