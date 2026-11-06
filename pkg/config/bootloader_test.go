package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinuxBootloaderToCmdLine(t *testing.T) {
	for _, test := range []struct {
		name       string
		initrdPath string
		want       []string
	}{
		{
			name: "without initrd",
			want: []string{"--kernel", "/vmlinux", "--kernel-cmdline", "console=hvc0"},
		},
		{
			name:       "with initrd",
			initrdPath: "/initrd",
			want:       []string{"--kernel", "/vmlinux", "--initrd", "/initrd", "--kernel-cmdline", "console=hvc0"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			bootloader := NewLinuxBootloader("/vmlinux", "console=hvc0", test.initrdPath)
			args, err := bootloader.ToCmdLine()
			require.NoError(t, err)
			require.Equal(t, test.want, args)
		})
	}
}
