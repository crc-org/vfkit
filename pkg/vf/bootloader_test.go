package vf

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/crc-org/vfkit/pkg/config"
	"github.com/stretchr/testify/require"
)

type isUncompressedCheckFunc func(t require.TestingT, value bool, msgAndArgs ...interface{})

type uncompressedKernelTest struct {
	filename            string
	isUncompressedCheck isUncompressedCheckFunc
}

var uncompressedKernelTests = map[string]uncompressedKernelTest{
	"fedora-amd64-compressed": {
		filename:            filepath.Join("testdata", "vmlinuz-truncated-6.4.11-200.fc38.x86_64"),
		isUncompressedCheck: require.False,
	},
	"fedora-arm64-compressed": {
		// this kernel is wrapped in an EFI binary, I don't know how to produce an uncompressed version
		filename:            filepath.Join("testdata", "vmlinuz-truncated-6.4.11-200.fc38.aarch64"),
		isUncompressedCheck: require.False,
	},
	"puipui-arm64-uncompressed": {
		filename:            filepath.Join("testdata", "vmlinux-truncated-0.1.0.puipui.aarch64"),
		isUncompressedCheck: require.True,
	},
	"puipui-am64-compressed": {
		filename:            filepath.Join("testdata", "vmlinux-truncated-0.1.0.puipui.x86_64"),
		isUncompressedCheck: require.False,
	},
	"rhel-arm64-uncompressed": {
		filename:            filepath.Join("testdata", "vmlinux-truncated-5.14.0-70.72.1.el9_0.aarch64"),
		isUncompressedCheck: require.True,
	},
	"rhel-arm64-compressed": {
		filename:            filepath.Join("testdata", "vmlinuz-truncated-5.14.0-70.72.1.el9_0.aarch64"),
		isUncompressedCheck: require.False,
	},
}

func TestUncompressedKernel(t *testing.T) {
	for name, test := range uncompressedKernelTests {
		t.Run(name, func(t *testing.T) {
			uncompressed, err := isKernelUncompressed(test.filename)
			require.NoError(t, err)
			test.isUncompressedCheck(t, uncompressed)
		})
	}
}

func TestLinuxBootloaderInitrd(t *testing.T) {
	kernelPath := filepath.Join("testdata", "vmlinux-truncated-0.1.0.puipui.aarch64")
	initrdPath := filepath.Join(t.TempDir(), "initrd")
	require.NoError(t, os.WriteFile(initrdPath, []byte("initrd test data"), 0600))

	for _, test := range []struct {
		name       string
		initrdPath string
		wantError  bool
	}{
		{name: "omitted"},
		{name: "supplied", initrdPath: initrdPath},
		{name: "missing", initrdPath: filepath.Join(t.TempDir(), "missing-initrd"), wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			bootloader, err := toVzLinuxBootloader(config.NewLinuxBootloader(kernelPath, "console=hvc0", test.initrdPath))
			if test.wantError {
				require.ErrorContains(t, err, "invalid initial RAM disk path")
				require.ErrorIs(t, err, os.ErrNotExist)
				return
			}
			require.NoError(t, err)
			require.Equal(t, fmt.Sprintf("vmlinuz: %q, initrd: %q, command-line: %q", kernelPath, test.initrdPath, "console=hvc0"), fmt.Sprint(bootloader))
		})
	}
}
