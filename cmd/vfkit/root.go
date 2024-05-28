package main

import (
	"fmt"
	"os"

	"github.com/crc-org/vfkit/pkg/cmdline"
	"github.com/onsi/gocleanup"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var opts = &cmdline.Options{}

var rootCmd = &cobra.Command{
	Use:   "vfkit",
	Short: "vfkit is a simple hypervisor using Apple's Virtualization framework",
	Long: `A hypervisor written in Go using Apple's Virtualization framework to run virtual machines.
                Complete documentation is available at https://github.com/crc-org/vfkit`,
	RunE: func(_ *cobra.Command, _ []string) error {
		if len(opts.LogLevel) > 0 {
			ll, err := getLogLevel()
			if err != nil {
				return err
			}
			logrus.SetLevel(ll)
		}
		vmConfig, err := newVMConfiguration(opts)
		if err != nil {
			return err
		}
		return runVFKit(vmConfig, opts)
	},
	Version: cmdline.Version(),
}

func init() {
	cmdline.AddFlags(rootCmd, opts)

	// this is almost the cobra default template with an added ':' before the version for crc's convenience
	versionTmpl := `{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version: %s" .Version}}
`
	rootCmd.SetVersionTemplate(versionTmpl)
}

func getLogLevel() (logrus.Level, error) {
	switch opts.LogLevel {
	case "error":
		return logrus.ErrorLevel, nil
	case "debug":
		return logrus.DebugLevel, nil
	case "info":
		return logrus.InfoLevel, nil
	}
	return 0, fmt.Errorf("unknown log level: %s", opts.LogLevel)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		gocleanup.Exit(1)
	}
}

func main() {
	Execute()
	gocleanup.Exit(0)
}
