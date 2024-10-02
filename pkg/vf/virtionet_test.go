package vf

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crc-org/vfkit/pkg/config"
	"github.com/stretchr/testify/require"
)

func testConnectUnixgram(t *testing.T) error {
	unixSocketPath := filepath.Join("/tmp", fmt.Sprintf("vnet-test-%x.sock", rand.Int31n(0xffff))) //#nosec G404 -- no need for crypto/rand here
	addr, err := net.ResolveUnixAddr("unixgram", unixSocketPath)
	require.NoError(t, err)

	l, err := net.ListenUnixgram("unixgram", addr)
	require.NoError(t, err)

	defer l.Close()
	defer os.Remove(unixSocketPath)

	dev := &VirtioNet{
		&config.VirtioNet{
			UnixSocketPath: unixSocketPath,
		},
		&net.UnixAddr{},
	}

	return dev.connectUnixPath()
}

func TestConnectUnixPath(t *testing.T) {
	t.Run("Successful connection - no error", func(t *testing.T) {
		err := testConnectUnixgram(t)
		require.NoError(t, err)
	})

	t.Run("Failed connection - End socket longer than 104 bytes", func(t *testing.T) {
		// Retrieve HOME env variable (used by the os.UserHomeDir)
		origUserHome := os.Getenv("HOME")
		defer func() {
			os.Setenv("HOME", origUserHome)
		}()

		// Create a string of 100 bytes to update the user home to be sure to create a socket path > 104 bytes
		b := bytes.Repeat([]byte("a"), 100)
		subDir := string(b)

		// Update HOME env so os.UserHomeDir returns the update path with subfolder
		updatedUserHome := filepath.Join(origUserHome, subDir)
		os.Setenv("HOME", updatedUserHome)
		defer os.RemoveAll(updatedUserHome)

		err := testConnectUnixgram(t)
		// It should return an error
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid argument")
	})
}

func TestLocalUnixSocketPath(t *testing.T) {
	t.Run("Success case - Creates temporary socket path", func(t *testing.T) {
		// Retrieve HOME env variable (used by the os.UserHomeDir)
		userHome := os.Getenv("HOME")

		path, err := localUnixSocketPath()

		// Assert successful execution
		require.NoError(t, err)

		// Check if path starts with the expected prefix
		expectedPrefix := filepath.Join(userHome, "Library", "Application Support", "vfkit")
		require.Truef(t, strings.HasPrefix(path, expectedPrefix), "Path doesn't start with expected prefix: %v", path)

		// Check if path ends with a socket extension
		require.Equalf(t, ".sock", filepath.Ext(path), "Path doesn't end with .sock extension: %v, ext is %v", path, filepath.Ext(path))
	})
}
