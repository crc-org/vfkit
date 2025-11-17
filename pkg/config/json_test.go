package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// This sets all the fields of the `obj` struct to non-empty values.
// This will be used to test JSON serialization as extensively as possible to
// avoid breaking backwards compatibility.
// `skipFields` can be used if there are some fields containing "magic" values
// which should not be overwritten with an arbitrary string, or if there are
// some fields with a type which `fillStruct` does not handle yet, and which
// are not interesting for serialization.
func fillStruct(t *testing.T, obj interface{}, skipFields []string) {
	val := reflect.ValueOf(obj).Elem()

	for _, e := range reflect.VisibleFields(val.Type()) {
		field := val.Type().FieldByIndex(e.Index)
		fieldVal := val.FieldByIndex(e.Index)
		typeName := val.Type().Name()

		if slices.Contains(skipFields, field.Name) {
			continue
		}
		switch fieldVal.Kind() {
		case reflect.Int, reflect.Int64:
			fieldVal.SetInt(2)
		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			fieldVal.SetUint(3)
		case reflect.Bool:
			fieldVal.SetBool(true)
		case reflect.String:
			fieldVal.SetString(field.Name)
		case reflect.Struct:
			// ignore the embedded struct, reflect.VisibleFields iterates over its fields
		case reflect.Slice:
			elemKind := fieldVal.Type().Elem().Kind()
			if elemKind != reflect.Uint8 {
				// SetBytes will panic on non-uint8 slices
				t.Fatalf("unsupported slice element kind '%s' for %s", elemKind, typeName)
			}
			fieldVal.SetBytes([]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55})
		default:
			t.Fatalf("unknown field kind '%s' for %s", fieldVal.Kind(), typeName)
		}
	}
}

type jsonTest struct {
	newVM        func(*testing.T) *VirtualMachine
	expectedJSON string
}

var jsonTests = map[string]jsonTest{
	"TestLinuxVM": {
		newVM:        newLinuxVM,
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"}}`,
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
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"},"timesync":{"vsockPort":1234}}`,
	},
	"TestIgnition": {
		newVM: func(t *testing.T) *VirtualMachine {
			vm := newLinuxVM(t)
			ignition, err := IgnitionNew("config", "socket")
			require.NoError(t, err)
			vm.Ignition = ignition
			return vm
		},
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"}, "ignition":{"kind":"ignition","configPath":"config","socketPath":"socket"}}`,
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
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"},"devices":[{"kind":"virtiorng"}]}`,
	},
	"TestVirtioBalloon": {
		newVM: func(t *testing.T) *VirtualMachine {
			vm := newLinuxVM(t)
			virtioBalloon, err := VirtioBalloonNew()
			require.NoError(t, err)
			err = vm.AddDevice(virtioBalloon)
			require.NoError(t, err)
			return vm
		},
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"},"devices":[{"kind":"virtioballoon"}]}`,
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
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"},"devices":[{"kind":"virtioblk","devName":"virtio-blk","imagePath":"/virtioblk1"},{"kind":"virtioblk","devName":"virtio-blk","imagePath":"/virtioblk2","deviceIdentifier":"virtio-blk2"}]}`,
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
			usb.SetReadOnly(true)
			// rosetta
			rosetta, err := RosettaShareNew("vz-rosetta")
			require.NoError(t, err)
			// NBD
			nbd, err := NetworkBlockDeviceNew("uri", 1, SynchronizationFullMode)
			require.NoError(t, err)
			err = vm.AddDevices(fs, usb, rosetta, nbd)
			require.NoError(t, err)

			return vm
		},
		expectedJSON: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"},"devices":[{"kind":"virtioserial","logFile":"/virtioserial"},{"kind":"virtioinput","inputType":"keyboard"},{"kind":"virtiogpu","usesGUI":false,"width":800,"height":600},{"kind":"virtionet","nat":true,"macAddress":"00:11:22:33:44:55"},{"kind":"virtiorng"},{"kind":"virtioblk","devName":"virtio-blk","imagePath":"/virtioblk"},{"kind":"virtiosock","port":1234,"socketURL":"/virtiovsock"},{"kind":"virtiofs","mountTag":"tag","sharedDir":"/virtiofs"},{"kind":"usbmassstorage","devName":"usb-mass-storage","imagePath":"/usbmassstorage","readOnly":true},{"kind":"rosetta","mountTag":"vz-rosetta","installRosetta":false,"ignoreIfMissing":false},{"kind":"nbd", "devName":"nbd", "uri":"uri", "DeviceIdentifier":"", "SynchronizationMode":"full","Timeout":1000000}]}`,
	},
}

type invalidJSONTest struct {
	json string
}

var invalidJSONTests = map[string]invalidJSONTest{
	"TestEmptyBootloaderKind": {
		json: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"empty",vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"}}`,
	},
	"TestInvalidBootloaderKind": {
		json: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"invalid",vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"}}`,
	},
	"TestMissingBootloaderKind": {
		json: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"}}`,
	},
	"TestEmptyDeviceKind": {
		json: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"},"devices":[{"kind":"","devName":"virtio-blk","imagePath":"/virtioblk1"}]}`,
	},
	"TestInvalidDeviceKind": {
		json: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"},"devices":[{"kind":"invalid","devName":"virtio-blk","imagePath":"/virtioblk1"}]}`,
	},
	"TestMissingDeviceKind": {
		json: `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"},"devices":[{"devName":"virtio-blk","imagePath":"/virtioblk1"}]}`,
	},
}

// These tests are there to ensure we don't change the JSON serializations of these objects by mistake.
// Adding new fields is fine, removing/renaming fields is not.
// This uses the `fillStruct` helper to set as many fields as possible to non-empty values.
// New types must be manually added to the tests.
var jsonStabilityTests = map[string]jsonStabilityTest{
	"VirtualMachine": {
		newObjectFunc: func(t *testing.T) any {
			vm := newLinuxVM(t)
			vm.Timesync = &TimeSync{VsockPort: 1234}
			vm.Devices = []VirtioDevice{&VirtioRng{}}

			return vm
		},
		skipFields:   []string{"Bootloader", "Devices", "Timesync", "Ignition", "Nested", "PidFile"},
		expectedJSON: `{"vcpus":3,"memoryBytes":3,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","kernelCmdLine":"console=hvc0","initrdPath":"/initrd"},"devices":[{"kind":"virtiorng"}],"timesync":{"vsockPort":1234}}`,
	},
	"RosettaShare": {
		obj:          &RosettaShare{},
		expectedJSON: `{"kind":"rosetta","mountTag":"MountTag","installRosetta":true,"ignoreIfMissing":true}`,
	},
	"VirtioFs": {
		obj:          &VirtioFs{},
		expectedJSON: `{"kind":"virtiofs","mountTag":"MountTag","sharedDir":"SharedDir"}`,
	},
	"VirtioGPU": {
		obj:          &VirtioGPU{},
		expectedJSON: `{"kind":"virtiogpu","usesGUI":true,"width":2,"height":2}`,
	},
	"VirtioNet": {
		obj:          &VirtioNet{},
		skipFields:   []string{"Socket"},
		expectedJSON: `{"kind":"virtionet","nat":true,"unixSocketPath":"UnixSocketPath","vfkitMagic":true,"macAddress":"00:11:22:33:44:55"}`,
	},
	"VirtioRNG": {
		obj:          &VirtioRng{},
		expectedJSON: `{"kind":"virtiorng"}`,
	},
	"VirtioSerial": {
		obj:          &VirtioSerial{},
		expectedJSON: `{"kind":"virtioserial","logFile":"LogFile","ptyName":"PtyName","usesPty":true,"usesStdio":true}`,
	},
	"VirtioVsock": {
		obj:          &VirtioVsock{},
		expectedJSON: `{"kind":"virtiosock","port":3,"socketURL":"SocketURL","listen":true}`,
	},
	"VirtioInput/keyboard": {
		newObjectFunc: func(t *testing.T) any {
			input, err := VirtioInputNew(VirtioInputKeyboardDevice)
			require.NoError(t, err)
			return input
		},
		skipFields:   []string{"InputType"},
		expectedJSON: `{"kind":"virtioinput","inputType":"keyboard"}`,
	},
	"VirtioInput/pointingDevice": {
		newObjectFunc: func(t *testing.T) any {
			input, err := VirtioInputNew(VirtioInputPointingDevice)
			require.NoError(t, err)
			return input
		},
		skipFields:   []string{"InputType"},
		expectedJSON: `{"kind":"virtioinput","inputType":"pointing"}`,
	},
	"VirtioBlk": {
		newObjectFunc: func(t *testing.T) any {
			blk, err := VirtioBlkNew("")
			require.NoError(t, err)
			blk.Type = DiskBackendImage
			return blk
		},

		skipFields:   []string{"DevName", "URI", "Type"},
		expectedJSON: `{"kind":"virtioblk","devName":"virtio-blk","imagePath":"ImagePath","readOnly":true,"type":"image","deviceIdentifier":"DeviceIdentifier"}`,
	},
	"USBMassStorage": {
		newObjectFunc: func(t *testing.T) any {
			usb, err := USBMassStorageNew("")
			require.NoError(t, err)
			usb.Type = DiskBackendImage
			return usb
		},
		skipFields:   []string{"DevName", "URI", "Type"},
		expectedJSON: `{"kind":"usbmassstorage","devName":"usb-mass-storage","imagePath":"ImagePath","readOnly":true,"type":"image"}`,
	},
	"NVMExpressController": {
		newObjectFunc: func(t *testing.T) any {
			nvme, err := NVMExpressControllerNew("")
			require.NoError(t, err)
			nvme.Type = DiskBackendImage
			return nvme
		},
		skipFields:   []string{"DevName", "URI", "Type"},
		expectedJSON: `{"kind":"nvme","devName":"nvme","imagePath":"ImagePath","readOnly":true,"type":"image"}`,
	},
	"LinuxBootloader": {
		obj:          &LinuxBootloader{},
		expectedJSON: `{"kind":"linuxBootloader","vmlinuzPath":"VmlinuzPath","kernelCmdLine":"KernelCmdLine","initrdPath":"InitrdPath"}`,
	},
	"EFIBootloader": {
		obj:          &EFIBootloader{},
		expectedJSON: `{"kind":"efiBootloader","efiVariableStorePath":"EFIVariableStorePath","createVariableStore":true}`,
	},
	"TimeSync": {
		obj:          &TimeSync{},
		expectedJSON: `{"vsockPort":3}`,
	},
	"NetworkBlockDevice": {
		newObjectFunc: func(t *testing.T) any {
			nbd, err := NetworkBlockDeviceNew("uri", 1000, SynchronizationFullMode)
			require.NoError(t, err)
			return nbd
		},
		skipFields:   []string{"DevName", "ImagePath"},
		expectedJSON: `{"kind":"nbd","DeviceIdentifier":"DeviceIdentifier","devName":"nbd","uri":"URI","readOnly":true,"SynchronizationMode":"SynchronizationMode","Timeout":2}`,
	},
}

type jsonStabilityTest struct {
	obj           any
	newObjectFunc func(*testing.T) any
	skipFields    []string
	expectedJSON  string
}

func testVirtioNetVfkitMagicJson(t *testing.T, vfkitMagic bool, useVfkitMagicDefault bool) {
	const (
		jsonVfkitMagicDefault = `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"},"devices":[{"kind":"virtionet","nat":false,"unixSocketPath":"/some/path/to/socket","macAddress":"00:11:22:33:44:55"}]}`
		jsonVfkitMagicFalse   = `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"},"devices":[{"kind":"virtionet","nat":false,"unixSocketPath":"/some/path/to/socket","macAddress":"00:11:22:33:44:55","vfkitMagic":false}]}`
		jsonVfkitMagicTrue    = `{"vcpus":3,"memoryBytes":4194304000,"bootloader":{"kind":"linuxBootloader","vmlinuzPath":"/vmlinuz","initrdPath":"/initrd","kernelCmdLine":"console=hvc0"},"devices":[{"kind":"virtionet","nat":false,"unixSocketPath":"/some/path/to/socket","macAddress":"00:11:22:33:44:55","vfkitMagic":true}]}`
	)
	var jsonStr string
	var unmarshalledVM VirtualMachine

	if useVfkitMagicDefault {
		jsonStr = jsonVfkitMagicDefault
	} else {
		if vfkitMagic {
			jsonStr = jsonVfkitMagicTrue
		} else {
			jsonStr = jsonVfkitMagicFalse
		}
	}
	err := json.Unmarshal([]byte(jsonStr), &unmarshalledVM)
	require.NoError(t, err)
	netDevs := unmarshalledVM.VirtioNetDevices()
	require.Len(t, netDevs, 1)
	require.Equal(t, vfkitMagic, netDevs[0].VfkitMagic)

	vm := newLinuxVM(t)
	dev, err := VirtioNetNew("00:11:22:33:44:55")
	require.NoError(t, err)
	dev.SetUnixSocketPath("/some/path/to/socket")
	if !useVfkitMagicDefault {
		dev.VfkitMagic = vfkitMagic
	}
	err = vm.AddDevice(dev)
	require.NoError(t, err)
	require.Equal(t, vfkitMagic, dev.VfkitMagic)

	require.Equal(t, *vm, unmarshalledVM)
}

func TestVirtioNetBackwardsCompat(t *testing.T) {
	/* Check that the vfkitMagic default is true when deserializing json */
	t.Run("VfkitMagicJsonDefault", func(t *testing.T) { testVirtioNetVfkitMagicJson(t, true, true) })

	/* Check that the vfkitMagic default can be overridden */
	t.Run("VfkitMagicJsonDefaultOverride", func(t *testing.T) { testVirtioNetVfkitMagicJson(t, false, false) })

	/* Check that explicitly setting vfkitMagic to true works as expected */
	t.Run("VfkitMagicJsonExplicitDefault", func(t *testing.T) { testVirtioNetVfkitMagicJson(t, true, false) })
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
		for name := range jsonStabilityTests {
			t.Run(fmt.Sprintf("Stability/%s", name), func(t *testing.T) {
				test := jsonStabilityTests[name]
				testJSONStability(t, &test)
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

func testJSONStability(t *testing.T, test *jsonStabilityTest) {
	obj := test.obj
	if obj == nil {
		obj = test.newObjectFunc(t)
	}
	fillStruct(t, obj, test.skipFields)
	data, err := json.Marshal(obj)
	require.NoError(t, err)
	fmt.Println(string(data))
	require.JSONEq(t, test.expectedJSON, string(data))
}

func newLinuxVM(*testing.T) *VirtualMachine {
	bootloader := NewLinuxBootloader("/vmlinuz", "console=hvc0", "/initrd")
	vm := NewVirtualMachine(3, 4_000, bootloader)

	return vm
}

func newUEFIVM(_ *testing.T) *VirtualMachine {
	bootloader := NewEFIBootloader("/variable-store", false)
	vm := NewVirtualMachine(3, 4_000, bootloader)

	return vm
}
