package main

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/crc-org/vfkit/pkg/cmdline"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartIgnitionProvisionerServer(t *testing.T) {
	socketPath := "virtiovsock"
	defer os.Remove(socketPath)

	ignitionData := []byte("ignition configuration")
	ignitionReader := bytes.NewReader(ignitionData)

	// Start the server using the socket so that it can returns the ignition data
	go func() {
		err := startIgnitionProvisionerServer(ignitionReader, socketPath)
		require.NoError(t, err)
	}()

	// Wait for the socket file to be created before serving, up to 2 seconds
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Make a request to the server
	client := http.Client{
		Transport: &http.Transport{
			Dial: func(_, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
	resp, err := client.Get("http://unix://" + socketPath)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify the response from the server is actually the ignition data
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, ignitionData, body)
}

func TestGenerateCloudInitImage(t *testing.T) {
	assetsDir, err := getTestAssetsDir()
	require.NoError(t, err)

	iso, err := generateCloudInitImage([]string{
		filepath.Join(assetsDir, "user-data"),
		filepath.Join(assetsDir, "meta-data"),
		filepath.Join(assetsDir, "network-config"),
	})
	require.NoError(t, err)

	assert.Contains(t, iso, "vfkit-cloudinit")

	_, err = os.Stat(iso)
	require.NoError(t, err)

	err = os.Remove(iso)
	require.NoError(t, err)
}

func TestGenerateCloudInitImageWithMissingFile(t *testing.T) {
	assetsDir, err := getTestAssetsDir()
	require.NoError(t, err)

	iso, err := generateCloudInitImage([]string{
		filepath.Join(assetsDir, "user-data"),
	})
	require.NoError(t, err)

	assert.Contains(t, iso, "vfkit-cloudinit")

	_, err = os.Stat(iso)
	require.NoError(t, err)

	err = os.Remove(iso)
	require.NoError(t, err)
}

func TestGenerateCloudInitImageWithWrongFile(t *testing.T) {
	assetsDir, err := getTestAssetsDir()
	require.NoError(t, err)

	iso, err := generateCloudInitImage([]string{
		filepath.Join(assetsDir, "seed.img"),
	})
	assert.Empty(t, iso)
	require.Error(t, err, "cloud-init needs user-data and meta-data files to work")
}

func TestGenerateCloudInitImageWithNoFile(t *testing.T) {
	iso, err := generateCloudInitImage([]string{})
	assert.Empty(t, iso)
	require.NoError(t, err)
}

func TestGUIAutoAddsGPUAndInput(t *testing.T) {
	opts := getTestVMOptions(t)
	opts.UseGUI = true

	vmConfig, err := newVMConfiguration(opts)
	require.NoError(t, err)
	require.NotNil(t, vmConfig)

	gpuDevices := vmConfig.VirtioGPUDevices()
	inputDevices := vmConfig.VirtioInputDevices()
	require.Len(t, gpuDevices, 1)
	assert.True(t, gpuDevices[0].UsesGUI)
	require.Len(t, inputDevices, 1)
	assert.Equal(t, inputDevices[0].InputType, config.VirtioInputKeyboardDevice)
}

func TestNoGUINoGPUAndInput(t *testing.T) {
	opts := getTestVMOptions(t)
	opts.UseGUI = false

	vmConfig, err := newVMConfiguration(opts)
	require.NoError(t, err)
	require.NotNil(t, vmConfig)

	gpuDevices := vmConfig.VirtioGPUDevices()
	require.Len(t, gpuDevices, 0)

	inputDevices := vmConfig.VirtioInputDevices()
	require.Len(t, inputDevices, 0)
}

func TestGUIWithExistingGPUAndNoInput(t *testing.T) {
	opts := getTestVMOptions(t)
	opts.UseGUI = true
	opts.Devices = []string{"virtio-gpu,width=1024,height=768"}

	vmConfig, err := newVMConfiguration(opts)
	require.NoError(t, err)
	require.NotNil(t, vmConfig)

	gpuDevices := vmConfig.VirtioGPUDevices()
	require.Len(t, gpuDevices, 1)

	inputDevices := vmConfig.VirtioInputDevices()
	require.Len(t, inputDevices, 1)
	assert.Equal(t, inputDevices[0].InputType, config.VirtioInputKeyboardDevice)
}

func TestGUIWithExistingGPUAndInput(t *testing.T) {
	opts := getTestVMOptions(t)
	opts.UseGUI = true
	opts.Devices = []string{"virtio-gpu,width=1024,height=768", "virtio-input,keyboard"}

	vmConfig, err := newVMConfiguration(opts)
	require.NoError(t, err)
	require.NotNil(t, vmConfig)

	gpuDevices := vmConfig.VirtioGPUDevices()
	require.Len(t, gpuDevices, 1)

	inputDevices := vmConfig.VirtioInputDevices()
	require.Len(t, inputDevices, 1)
	assert.Equal(t, inputDevices[0].InputType, config.VirtioInputKeyboardDevice)
}

func TestGUIWithExistingInputAndNoGPU(t *testing.T) {
	opts := getTestVMOptions(t)
	opts.UseGUI = true
	opts.Devices = []string{"virtio-input,keyboard"}

	vmConfig, err := newVMConfiguration(opts)
	require.NoError(t, err)
	require.NotNil(t, vmConfig)

	inputDevices := vmConfig.VirtioInputDevices()
	require.Len(t, inputDevices, 1)
	assert.Equal(t, inputDevices[0].InputType, config.VirtioInputKeyboardDevice)

	gpuDevices := vmConfig.VirtioGPUDevices()
	require.Len(t, gpuDevices, 1)
	assert.True(t, gpuDevices[0].UsesGUI)
}

func getTestAssetsDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	projectRoot := filepath.Join(currentDir, "../../")
	return filepath.Join(projectRoot, "test", "assets"), nil
}

func getTestVMOptions(t *testing.T) *cmdline.Options {
	opts := &cmdline.Options{
		Vcpus:     1,
		MemoryMiB: 512,
		Devices:   []string{},
	}
	err := opts.Bootloader.Set("efi")
	require.NoError(t, err)
	return opts
}
