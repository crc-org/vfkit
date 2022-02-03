package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const vfkitVersion = "0.0.1"

var opts = &cmdlineOptions{}

var rootCmd = &cobra.Command{
	Use:   "vfkit",
	Short: "vfkit is a simple hypervisor using Apple's virtualization framework",
	Long: `A hypervisor written in Go using Apple's virtualization framework to run linux virtual machines.
                Complete documentation is available at https://github.com/code-ready/vfkit`,
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
	rootCmd.Flags().StringVarP(&opts.vmlinuzPath, "kernel", "k", "", "path to the virtual machine linux kernel")
	rootCmd.Flags().StringVarP(&opts.kernelCmdline, "kernel-cmdline", "C", "", "linux kernel command line")
	rootCmd.Flags().StringVarP(&opts.initrdPath, "initrd", "i", "", "path to the virtual machine initrd")
	rootCmd.MarkFlagRequired("kernel")
	rootCmd.MarkFlagRequired("kernel-cmdline")
	rootCmd.MarkFlagRequired("initrd")

	rootCmd.Flags().UintVarP(&opts.vcpus, "cpus", "c", 1, "number of virtual CPUs")
	// FIXME: use go-units for parsing
	rootCmd.Flags().UintVarP(&opts.memoryMiB, "memory", "m", 512, "virtual machine RAM size in mibibytes")

	rootCmd.Flags().StringArrayVarP(&opts.devices, "device", "d", []string{}, "devices")

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
