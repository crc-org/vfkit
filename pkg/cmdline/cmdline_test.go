package cmdline

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func validateFlags(args ...string) error {
	cmd := &cobra.Command{}
	AddFlags(cmd, &Options{})
	if err := cmd.ParseFlags(args); err != nil {
		return err
	}
	return cmd.ValidateFlagGroups()
}

func TestLegacyKernelFlags(t *testing.T) {
	require.NoError(t, validateFlags("--kernel", "vmlinuz", "--kernel-cmdline", "console=hvc0"))
	require.NoError(t, validateFlags("--kernel", "vmlinuz", "--kernel-cmdline", "console=hvc0", "--initrd", "initrd.img"))
	require.Error(t, validateFlags("--kernel", "vmlinuz"))
	require.Error(t, validateFlags("--kernel", "vmlinuz", "--initrd", "initrd.img"))
	require.Error(t, validateFlags("--initrd", "initrd.img", "--bootloader", "efi,variable-store=vars"))
}
