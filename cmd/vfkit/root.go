package main

import (
	"fmt"
	"os"

	"github.com/crc-org/vfkit/pkg/cmdline"
	"github.com/spf13/cobra"
)

const vfkitVersion = "0.0.4"

var opts = &cmdline.Options{}

var rootCmd = &cobra.Command{
	Use:   "vfkit",
	Short: "vfkit is a simple hypervisor using Apple's virtualization framework",
	Long: `A hypervisor written in Go using Apple's virtualization framework to run linux virtual machines.
                Complete documentation is available at https://github.com/crc-org/vfkit`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vmConfig, err := newVMConfiguration(opts)
		if err != nil {
			return err
		}
		return runVirtualMachine(vmConfig)
	},
	Version: vfkitVersion,
}

func init() {
	cmdline.AddFlags(rootCmd, opts)

	// this is almost the cobra default template with an added ':' before the version for crc's convenience
	versionTmpl := `{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version: %s" .Version}}
`
	rootCmd.SetVersionTemplate(versionTmpl)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
