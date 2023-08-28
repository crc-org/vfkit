package vf

import (
	"path/filepath"
	"testing"

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
