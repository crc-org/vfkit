package main

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"testing"

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
