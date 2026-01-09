package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type virtioDevTest struct {
	newDev           func() (VirtioDevice, error)
	expectedDev      VirtioDevice
	expectedCmdLine  []string
	alternateCmdLine []string
	errorMsg         string
}

func getTestVirtioBlkDevice(testImagePath string) (*VirtioBlk, error) {
	err := os.WriteFile(testImagePath, []byte{'0', '0', '0', '0'}, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to write test image: %v", err)
	}
	return VirtioBlkNew(testImagePath)
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

func testErrorVirtioDev(t *testing.T, test *virtioDevTest) {
	dev, err := test.newDev()
	if err != nil {
		require.EqualError(t, err, test.errorMsg)
		return
	}

	_, err = dev.ToCmdLine()
	require.Error(t, err)
	require.EqualError(t, err, test.errorMsg)
}

func TestVirtioDevices(t *testing.T) {
	testImagePath := filepath.Join(t.TempDir(), "test.img")
	var virtioDevTests = map[string]virtioDevTest{
		"NewVirtioBlk": {
			newDev: func() (VirtioDevice, error) {
				return getTestVirtioBlkDevice(testImagePath)
			},
			expectedDev: &VirtioBlk{
				DiskStorageConfig: DiskStorageConfig{
					StorageConfig: StorageConfig{
						DevName: "virtio-blk",
					},
					ImagePath: testImagePath,
				},
				DeviceIdentifier: "",
			},
			expectedCmdLine: []string{"--device", fmt.Sprintf("virtio-blk,path=%s", testImagePath)},
		},
		"NewVirtioBlkWithDevId": {
			newDev: func() (VirtioDevice, error) {
				dev, err := getTestVirtioBlkDevice(testImagePath)
				if err != nil {
					return nil, err
				}
				dev.SetDeviceIdentifier("test")
				return dev, nil
			},
			expectedDev: &VirtioBlk{
				DiskStorageConfig: DiskStorageConfig{
					StorageConfig: StorageConfig{
						DevName: "virtio-blk",
					},
					ImagePath: testImagePath,
				},
				DeviceIdentifier: "test",
			},
			expectedCmdLine:  []string{"--device", fmt.Sprintf("virtio-blk,path=%s,deviceId=test", testImagePath)},
			alternateCmdLine: []string{"--device", fmt.Sprintf("virtio-blk,deviceId=test,path=%s", testImagePath)},
		},
		"NewVirtioBlkWithType": {
			newDev: func() (VirtioDevice, error) {
				dev, err := getTestVirtioBlkDevice(testImagePath)
				if err != nil {
					return nil, err
				}
				dev.Type = DiskBackendBlockDevice
				return dev, nil
			},
			expectedDev: &VirtioBlk{
				DiskStorageConfig: DiskStorageConfig{
					StorageConfig: StorageConfig{
						DevName: "virtio-blk",
					},
					ImagePath: testImagePath,
					Type:      DiskBackendBlockDevice,
				},
				DeviceIdentifier: "",
			},
			expectedCmdLine:  []string{"--device", fmt.Sprintf("virtio-blk,path=%s,type=dev", testImagePath)},
			alternateCmdLine: []string{"--device", fmt.Sprintf("virtio-blk,type=dev,path=%s", testImagePath)},
		},
		"NewVirtioBlkWithDefaultType": {
			newDev: func() (VirtioDevice, error) {
				dev, err := getTestVirtioBlkDevice(testImagePath)
				if err != nil {
					return nil, err
				}
				dev.Type = DiskBackendDefault
				return dev, nil
			},
			expectedDev: &VirtioBlk{
				DiskStorageConfig: DiskStorageConfig{
					StorageConfig: StorageConfig{
						DevName: "virtio-blk",
					},
					ImagePath: testImagePath,
					Type:      DiskBackendDefault,
				},
				DeviceIdentifier: "",
			},
			expectedCmdLine: []string{"--device", fmt.Sprintf("virtio-blk,path=%s", testImagePath)},
		},
		"NewNVMe": {
			newDev: func() (VirtioDevice, error) { return NVMExpressControllerNew("/foo/bar") },
			expectedDev: &NVMExpressController{
				DiskStorageConfig: DiskStorageConfig{
					StorageConfig: StorageConfig{
						DevName: "nvme",
					},
					ImagePath: "/foo/bar",
				},
			},
			expectedCmdLine: []string{"--device", "nvme,path=/foo/bar"},
		},
		"NewNVMeWithType": {
			newDev: func() (VirtioDevice, error) {
				dev, err := NVMExpressControllerNew("/foo/bar")
				if err != nil {
					return nil, err
				}
				dev.Type = DiskBackendImage
				return dev, nil
			},
			expectedDev: &NVMExpressController{
				DiskStorageConfig: DiskStorageConfig{
					StorageConfig: StorageConfig{
						DevName: "nvme",
					},
					ImagePath: "/foo/bar",
					Type:      DiskBackendImage,
				},
			},
			expectedCmdLine:  []string{"--device", "nvme,path=/foo/bar,type=image"},
			alternateCmdLine: []string{"--device", "nvme,type=image,path=/foo/bar"},
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
				DirectorySharingConfig: DirectorySharingConfig{
					MountTag: "myTag",
				},
			},
			expectedCmdLine:  []string{"--device", "virtio-fs,sharedDir=/foo/bar,mountTag=myTag"},
			alternateCmdLine: []string{"--device", "virtio-fs,mountTag=myTag,sharedDir=/foo/bar"},
		},
		"NewRosettaShare": {
			newDev: func() (VirtioDevice, error) { return RosettaShareNew("myTag") },
			expectedDev: &RosettaShare{
				DirectorySharingConfig: DirectorySharingConfig{
					MountTag: "myTag",
				},
			},
			expectedCmdLine: []string{"--device", "rosetta,mountTag=myTag"},
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
		"NewVirtioSerialPty": {
			newDev: VirtioSerialNewPty,
			expectedDev: &VirtioSerial{
				UsesPty: true,
			},
			expectedCmdLine: []string{"--device", "virtio-serial,pty"},
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
				VfkitMagic:     true,
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
				DiskStorageConfig: DiskStorageConfig{
					StorageConfig: StorageConfig{
						DevName: "usb-mass-storage",
					},
					ImagePath: "/foo/bar",
				},
			},
			expectedCmdLine: []string{"--device", "usb-mass-storage,path=/foo/bar"},
		},
		"NewUSBMassStorageReadOnly": {
			newDev: func() (VirtioDevice, error) {
				dev, err := USBMassStorageNew("/foo/bar")
				if err != nil {
					return nil, err
				}
				dev.SetReadOnly(true)
				return dev, err
			},
			expectedDev: &USBMassStorage{
				DiskStorageConfig: DiskStorageConfig{
					StorageConfig: StorageConfig{
						DevName:  "usb-mass-storage",
						ReadOnly: true,
					},
					ImagePath: "/foo/bar",
				},
			},
			expectedCmdLine: []string{"--device", "usb-mass-storage,path=/foo/bar,readonly"},
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
				VirtioGPUResolution{Width: 800, Height: 600},
			},
			expectedCmdLine: []string{"--device", "virtio-gpu,width=800,height=600"},
		},
		"NewVirtioGPUDeviceWithDimensions": {
			newDev: func() (VirtioDevice, error) {
				dev, err := VirtioGPUNew()
				if err != nil {
					return nil, err
				}
				dev.(*VirtioGPU).VirtioGPUResolution = VirtioGPUResolution{Width: 1920, Height: 1080}
				return dev, nil
			},
			expectedDev: &VirtioGPU{
				false,
				VirtioGPUResolution{Width: 1920, Height: 1080},
			},
			expectedCmdLine: []string{"--device", "virtio-gpu,width=1920,height=1080"},
		},
		"NetworkBlockDeviceNew": {
			newDev: func() (VirtioDevice, error) {
				return NetworkBlockDeviceNew("nbd://1.1.1.1:10000", 1000, SynchronizationNoneMode)
			},
			expectedDev: &NetworkBlockDevice{
				NetworkBlockStorageConfig: NetworkBlockStorageConfig{
					StorageConfig: StorageConfig{
						DevName: "nbd",
					},
					URI: "nbd://1.1.1.1:10000",
				},
				DeviceIdentifier:    "",
				Timeout:             time.Duration(1000 * time.Millisecond),
				SynchronizationMode: SynchronizationNoneMode,
			},
			expectedCmdLine: []string{"--device", "nbd,uri=nbd://1.1.1.1:10000,timeout=1000,sync=none"},
		},
		"NewVirtioBalloon": {
			newDev:          VirtioBalloonNew,
			expectedDev:     &VirtioBalloon{},
			expectedCmdLine: []string{"--device", "virtio-balloon"},
		},
		"VirtioNetWithVfkitMagicOff": {
			newDev: func() (VirtioDevice, error) {
				dev := &VirtioNet{
					UnixSocketPath: "/tmp/test.sock",
					VfkitMagic:     false,
				}
				return dev, nil
			},
			expectedDev: &VirtioNet{
				UnixSocketPath: "/tmp/test.sock",
				VfkitMagic:     false,
			},
			expectedCmdLine: []string{"--device", "virtio-net,type=unixgram,path=/tmp/test.sock,vfkitMagic=off"},
		},
		"VirtioNetWithVfkitMagicOn": {
			newDev: func() (VirtioDevice, error) {
				dev := &VirtioNet{
					UnixSocketPath: "/tmp/test.sock",
					VfkitMagic:     true,
				}
				return dev, nil
			},
			expectedDev: &VirtioNet{
				UnixSocketPath: "/tmp/test.sock",
				VfkitMagic:     true,
			},
			expectedCmdLine: []string{"--device", "virtio-net,unixSocketPath=/tmp/test.sock"},
		},
		"VirtioNetDefaultVfkitMagic": {
			newDev: func() (VirtioDevice, error) {
				return deviceFromCmdLine("virtio-net,type=unixgram,path=/tmp/default.sock")
			},
			expectedDev: &VirtioNet{
				UnixSocketPath: "/tmp/default.sock",
				VfkitMagic:     true,
			},
			expectedCmdLine: []string{"--device", "virtio-net,unixSocketPath=/tmp/default.sock"},
		},
		"VirtioNetUnixSocketPath": {
			newDev: func() (VirtioDevice, error) {
				return deviceFromCmdLine("virtio-net,unixSocketPath=/tmp/socket.sock")
			},
			expectedDev: &VirtioNet{
				UnixSocketPath: "/tmp/socket.sock",
				VfkitMagic:     true,
			},
			expectedCmdLine: []string{"--device", "virtio-net,unixSocketPath=/tmp/socket.sock"},
		},
		"VirtioNetUnixSocketPathWithVfkitMagicOff": {
			newDev: func() (VirtioDevice, error) {
				return deviceFromCmdLine("virtio-net,unixSocketPath=/tmp/socket.sock,vfkitMagic=off")
			},
			expectedDev: &VirtioNet{
				UnixSocketPath: "/tmp/socket.sock",
				VfkitMagic:     false,
			},
			expectedCmdLine: []string{"--device", "virtio-net,type=unixgram,path=/tmp/socket.sock,vfkitMagic=off"},
		},
		"VirtioNetVfkitMagicInvalidValue": {
			newDev: func() (VirtioDevice, error) {
				return deviceFromCmdLine("virtio-net,unixSocketPath=/tmp/test.sock,vfkitMagic=foo")
			},
			errorMsg: "invalid value for vfkitMagic: foo (expected on/off)",
		},
		"VirtioNetInvalidTypeFoo": {
			newDev: func() (VirtioDevice, error) {
				return deviceFromCmdLine("virtio-net,type=foo")
			},
			errorMsg: "unsupported virtio-net type: foo (only 'unixgram' is supported)",
		},
		"VirtioNetTypeWithoutPath": {
			newDev: func() (VirtioDevice, error) {
				return deviceFromCmdLine("virtio-net,type=unixgram")
			},
			errorMsg: "'type' option requires 'path' to be specified",
		},
		"VirtioNetOffloadingInvalidValueOn": {
			newDev: func() (VirtioDevice, error) {
				return deviceFromCmdLine("virtio-net,type=unixgram,path=/tmp/test.sock,offloading=on")
			},
			errorMsg: "invalid value for offloading: on (only 'off' is supported)",
		},
		"VirtioNetOffloadingOff": {
			newDev: func() (VirtioDevice, error) {
				return deviceFromCmdLine("virtio-net,type=unixgram,path=/tmp/test.sock,offloading=off")
			},
			expectedDev: &VirtioNet{
				UnixSocketPath: "/tmp/test.sock",
				VfkitMagic:     true,
			},
			expectedCmdLine: []string{"--device", "virtio-net,unixSocketPath=/tmp/test.sock"},
		},
	}
	t.Run("virtio-devices", func(t *testing.T) {
		for name := range virtioDevTests {
			t.Run(name, func(t *testing.T) {
				test := virtioDevTests[name]
				if test.errorMsg != "" {
					testErrorVirtioDev(t, &test)
				} else {
					testVirtioDev(t, &test)
				}
			})
		}
	})
}
