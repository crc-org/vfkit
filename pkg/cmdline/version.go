package cmdline

// set using the '-X github.com/crc-org/vfkit/pkg/cmdline.gitVersion' linker flag
var gitVersion = "unknown"

func Version() string {
	return gitVersion
}
