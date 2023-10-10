package test

// The code in this file is heavily based on https://github.com/Code-Hex/vz/blob/40946b951fffa07406b272c582eda76de7c24028/osversion.go#L46-L70

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/mod/semver"
)

func macOSAvailable(version float64) error {
	if macOSMajorMinorVersion() < version {
		return ErrUnsupportedOSVersion
	}
	return nil
}

var (
	// ErrUnsupportedOSVersion is returned when calling a method which is only
	// available in newer macOS versions.
	ErrUnsupportedOSVersion = errors.New("unsupported macOS version")

	majorMinorVersion     float64
	majorMinorVersionOnce interface{ Do(func()) } = &sync.Once{}
)

func fetchMajorMinorVersion() (float64, error) {
	osver, err := syscall.Sysctl("kern.osproductversion")
	if err != nil {
		return 0, err
	}
	prefix := "v"
	majorMinor := strings.TrimPrefix(semver.MajorMinor(prefix+osver), prefix)
	version, err := strconv.ParseFloat(majorMinor, 64)
	if err != nil {
		return 0, err
	}
	return version, nil
}

func macOSMajorMinorVersion() float64 {
	majorMinorVersionOnce.Do(func() {
		version, err := fetchMajorMinorVersion()
		if err != nil {
			panic(err)
		}
		majorMinorVersion = version
	})
	return majorMinorVersion
}
