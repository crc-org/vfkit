package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type virtioDevTest struct {
	newDev           func() (VirtioDevice, error)
	expectedDev      VirtioDevice
	expectedCmdLine  []string
	alternateCmdLine []string
}

var virtioDevTests = map[string]virtioDevTest{
	"NewVirtioBlk": {
		newDev: func() (VirtioDevice, error) { return VirtioBlkNew("/foo/bar") },
		expectedDev: &VirtioBlk{
			StorageConfig: StorageConfig{
				DevName:   "virtio-blk",
				ImagePath: "/foo/bar",
			},
			DeviceIdentifier: "",
		},
		expectedCmdLine: []string{"--device", "virtio-blk,path=/foo/bar"},
	},
	"NewVirtioBlkWithDevId": {
		newDev: func() (VirtioDevice, error) {
			dev, err := VirtioBlkNew("/foo/bar")
			if err != nil {
				return nil, err
			}
			dev.SetDeviceIdentifier("test")
			return dev, nil
		},
		expectedDev: &VirtioBlk{
			StorageConfig: StorageConfig{
				DevName:   "virtio-blk",
				ImagePath: "/foo/bar",
			},
			DeviceIdentifier: "test",
		},
		expectedCmdLine:  []string{"--device", "virtio-blk,path=/foo/bar,deviceId=test"},
		alternateCmdLine: []string{"--device", "virtio-blk,deviceId=test,path=/foo/bar"},
	},
	"NewVirtioFs": {
		newDev: func() (VirtioDevice, error) { return VirtioFsNew("/foo/bar", "") },
		expectedDev: &VirtioFs{
			SharedDir: "/foo/bar",
		},
		expectedCmdLine: []string{"--device", "virtio-fs,sharedDir=/foo/bar"},
	},
	"NewVirtioFsWithTag": {
		newDev: func() (VirtioDevice, error) { return VirtioFsNew("/foo/bar", "myTag") },
		expectedDev: &VirtioFs{
			SharedDir: "/foo/bar",
			MountTag:  "myTag",
		},
		expectedCmdLine:  []string{"--device", "virtio-fs,sharedDir=/foo/bar,mountTag=myTag"},
		alternateCmdLine: []string{"--device", "virtio-fs,mountTag=myTag,sharedDir=/foo/bar"},
	},
	"NewVirtioVsock": {
		newDev: func() (VirtioDevice, error) { return VirtioVsockNew(1234, "/foo/bar.unix", false) },
		expectedDev: &VirtioVsock{
			Port:      1234,
			SocketURL: "/foo/bar.unix",
		},
		expectedCmdLine:  []string{"--device", "virtio-vsock,port=1234,socketURL=/foo/bar.unix,connect"},
		alternateCmdLine: []string{"--device", "virtio-vsock,socketURL=/foo/bar.unix,connect,port=1234"},
	},
	"NewVirtioVsockWithListen": {
		newDev: func() (VirtioDevice, error) { return VirtioVsockNew(1234, "/foo/bar.unix", true) },
		expectedDev: &VirtioVsock{
			Port:      1234,
			SocketURL: "/foo/bar.unix",
			Listen:    true,
		},
		expectedCmdLine:  []string{"--device", "virtio-vsock,port=1234,socketURL=/foo/bar.unix,listen"},
		alternateCmdLine: []string{"--device", "virtio-vsock,socketURL=/foo/bar.unix,listen,port=1234"},
	},
	"NewVirtioRng": {
		newDev:          VirtioRngNew,
		expectedDev:     &VirtioRng{},
		expectedCmdLine: []string{"--device", "virtio-rng"},
	},
	"NewVirtioSerial": {
		newDev: func() (VirtioDevice, error) { return VirtioSerialNew("/foo/bar.log") },
		expectedDev: &VirtioSerial{
			LogFile: "/foo/bar.log",
		},
		expectedCmdLine: []string{"--device", "virtio-serial,logFilePath=/foo/bar.log"},
	},
	"NewVirtioSerialStdio": {
		newDev: VirtioSerialNewStdio,
		expectedDev: &VirtioSerial{
			UsesStdio: true,
		},
		expectedCmdLine: []string{"--device", "virtio-serial,stdio"},
	},
	"NewVirtioNet": {
		newDev: func() (VirtioDevice, error) { return VirtioNetNew("") },
		expectedDev: &VirtioNet{
			Nat: true,
		},
		expectedCmdLine: []string{"--device", "virtio-net,nat"},
	},
	"NewVirtioNetWithPath": {
		newDev: func() (VirtioDevice, error) {
			dev, err := VirtioNetNew("")
			if err != nil {
				return nil, err
			}
			dev.SetUnixSocketPath("/tmp/unix.sock")
			return dev, nil
		},
		expectedDev: &VirtioNet{
			Nat:            false,
			UnixSocketPath: "/tmp/unix.sock",
		},
		expectedCmdLine: []string{"--device", "virtio-net,unixSocketPath=/tmp/unix.sock"},
	},
	"NewVirtioNetWithMacAddress": {
		newDev: func() (VirtioDevice, error) { return VirtioNetNew("00:11:22:33:44:55") },
		expectedDev: &VirtioNet{
			Nat:        true,
			MacAddress: []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		},
		expectedCmdLine:  []string{"--device", "virtio-net,nat,mac=00:11:22:33:44:55"},
		alternateCmdLine: []string{"--device", "virtio-net,mac=00:11:22:33:44:55,nat"},
	},
	"NewUSBMassStorage": {
		newDev: func() (VirtioDevice, error) { return USBMassStorageNew("/foo/bar") },
		expectedDev: &USBMassStorage{
			StorageConfig: StorageConfig{
				DevName:   "usb-mass-storage",
				ImagePath: "/foo/bar",
			},
		},
		expectedCmdLine: []string{"--device", "usb-mass-storage,path=/foo/bar"},
	},
	"NewVirtioInputWithPointingDevice": {
		newDev: func() (VirtioDevice, error) { return VirtioInputNew("pointing") },
		expectedDev: &VirtioInput{
			InputType: "pointing",
		},
		expectedCmdLine: []string{"--device", "virtio-input,pointing"},
	},
	"NewVirtioInputWithKeyboardDevice": {
		newDev: func() (VirtioDevice, error) { return VirtioInputNew("keyboard") },
		expectedDev: &VirtioInput{
			InputType: "keyboard",
		},
		expectedCmdLine: []string{"--device", "virtio-input,keyboard"},
	},
	"NewVirtioGPUDevice": {
		newDev: VirtioGPUNew,
		expectedDev: &VirtioGPU{
			false,
			VirtioGPUResolution{800, 600},
		},
		expectedCmdLine: []string{"--device", "virtio-gpu,height=800,width=600"},
	},
	"NewVirtioGPUDeviceWithDimensions": {
		newDev: func() (VirtioDevice, error) {
			dev, err := VirtioGPUNew()
			if err != nil {
				return nil, err
			}
			dev.(*VirtioGPU).VirtioGPUResolution = VirtioGPUResolution{1920, 1080}
			return dev, nil
		},
		expectedDev: &VirtioGPU{
			false,
			VirtioGPUResolution{1920, 1080},
		},
		expectedCmdLine: []string{"--device", "virtio-gpu,height=1920,width=1080"},
	},
}

func testVirtioDev(t *testing.T, test *virtioDevTest) {
	dev, err := test.newDev()
	require.NoError(t, err)
	assert.Equal(t, dev, test.expectedDev)

	cmdLine, err := dev.ToCmdLine()
	require.NoError(t, err)
	assert.Equal(t, cmdLine, test.expectedCmdLine)

	dev, err = deviceFromCmdLine(cmdLine[1])
	require.NoError(t, err)

	assert.Equal(t, dev, test.expectedDev)

	if test.alternateCmdLine == nil {
		return
	}

	dev, err = deviceFromCmdLine(test.alternateCmdLine[1])
	require.NoError(t, err)
	assert.Equal(t, dev, test.expectedDev)
	cmdLine, err = dev.ToCmdLine()
	require.NoError(t, err)
	assert.Equal(t, cmdLine, test.expectedCmdLine)

}

func TestVirtioDevices(t *testing.T) {
	t.Run("virtio-devices", func(t *testing.T) {
		for name := range virtioDevTests {
			t.Run(name, func(t *testing.T) {
				test := virtioDevTests[name]
				testVirtioDev(t, &test)
			})
		}

	})
}
