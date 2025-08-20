package cmdline

import (
	"github.com/spf13/cobra"
)

type Options struct {
	Vcpus     uint
	MemoryMiB uint

	VmlinuzPath   string
	KernelCmdline string
	InitrdPath    string

	Bootloader stringSliceValue

	TimeSync string

	Devices []string

	RestfulURI string

	LogLevel string

	UseGUI bool

	IgnitionPath string

	CloudInitFiles stringSliceValue

	Nested bool

	PidFile string
}

const DefaultRestfulURI = "none://"

func AddFlags(cmd *cobra.Command, opts *Options) {
	cmd.Flags().StringVarP(&opts.VmlinuzPath, "kernel", "k", "", "path to the virtual machine Linux kernel")
	cmd.Flags().StringVarP(&opts.KernelCmdline, "kernel-cmdline", "C", "", "Linux kernel command line")
	cmd.Flags().StringVarP(&opts.InitrdPath, "initrd", "i", "", "path to the virtual machine initrd")

	cmd.Flags().VarP(&opts.Bootloader, "bootloader", "b", "bootloader configuration")
	cmd.Flags().BoolVar(&opts.UseGUI, "gui", false, "display the contents of the virtual machine onto a graphical user interface")

	cmd.MarkFlagsMutuallyExclusive("kernel", "bootloader")
	cmd.MarkFlagsMutuallyExclusive("initrd", "bootloader")
	cmd.MarkFlagsMutuallyExclusive("kernel-cmdline", "bootloader")
	cmd.MarkFlagsRequiredTogether("kernel", "initrd", "kernel-cmdline")

	cmd.Flags().UintVarP(&opts.Vcpus, "cpus", "c", 1, "number of virtual CPUs")
	// FIXME: use go-units for parsing
	cmd.Flags().UintVarP(&opts.MemoryMiB, "memory", "m", 512, "virtual machine RAM size in mibibytes")

	cmd.Flags().StringVarP(&opts.TimeSync, "timesync", "t", "", "sync guest time when host wakes up from sleep")
	cmd.Flags().StringArrayVarP(&opts.Devices, "device", "d", []string{}, "devices")

	cmd.Flags().StringVar(&opts.LogLevel, "log-level", "", "set log level")
	cmd.Flags().StringVar(&opts.RestfulURI, "restful-uri", DefaultRestfulURI, "URI address for RESTful services")

	cmd.Flags().StringVar(&opts.IgnitionPath, "ignition", "", "path to the ignition file")
	cmd.Flags().VarP(&opts.CloudInitFiles, "cloud-init", "", "path to user-data and meta-data cloud-init configuration files")
	cmd.Flags().BoolVarP(&opts.Nested, "nested", "n", false, "enable nested virtualization")
	cmd.Flags().StringVar(&opts.PidFile, "pidfile", "", "path to the pid file")
}
