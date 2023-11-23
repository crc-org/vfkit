package cmdline

import (
	"runtime/debug"
)

// set using the '-X github.com/crc-org/vfkit/pkg/cmdline.gitVersion' linker flag
var gitVersion = "unknown"

func Version() string {
	switch {
	// This will be set when building from git using make
	case gitVersion != "":
		return gitVersion
	// moduleVersionFromBuildInfo() will be set when using `go install`
	default:
		return moduleVersionFromBuildInfo()
	}
}

func moduleVersionFromBuildInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	if info.Main.Version == "(devel)" {
		return ""
	}
	return info.Main.Version
}
