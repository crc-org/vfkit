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

func getTestAssetsDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	projectRoot := filepath.Join(currentDir, "../../")
	return filepath.Join(projectRoot, "test", "assets"), nil
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
