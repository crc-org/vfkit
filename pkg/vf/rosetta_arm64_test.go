package vf

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Code-Hex/vz/v3"
	"github.com/crc-org/vfkit/pkg/config"
	"github.com/stretchr/testify/require"
)

type checkRosettaAvailabilityTest struct {
	installRosetta                         bool
	ignoreIfMissing                        bool
	checkRosettaDirectoryShareAvailability func() vz.LinuxRosettaAvailability
	doInstallRosetta                       func() error
	errorValue                             string
}

var checkRosettaAvailabilityTests = map[string]checkRosettaAvailabilityTest{
	"TestRosettaIsNotSupported": {
		checkRosettaDirectoryShareAvailability: func() vz.LinuxRosettaAvailability {
			return vz.LinuxRosettaAvailabilityNotSupported
		},
		errorValue: "rosetta is not supported",
	},
	"TestRosettaInstalled": {
		checkRosettaDirectoryShareAvailability: func() vz.LinuxRosettaAvailability {
			return vz.LinuxRosettaAvailabilityInstalled
		},
	},
	"TestRosettaNotInstalled-NotToBeInstalled": {
		installRosetta: false,
		checkRosettaDirectoryShareAvailability: func() vz.LinuxRosettaAvailability {
			return vz.LinuxRosettaAvailabilityNotInstalled
		},
		errorValue: "rosetta is not installed",
	},
	"TestRosettaNotInstalled-InstallationCancelledButIgnoreIfMissingFalse": {
		installRosetta:  true,
		ignoreIfMissing: false,
		checkRosettaDirectoryShareAvailability: func() vz.LinuxRosettaAvailability {
			return vz.LinuxRosettaAvailabilityNotInstalled
		},
		doInstallRosetta: func() error {
			return fmt.Errorf("VZErrorDomain Code=%d", vz.ErrorOperationCancelled)
		},
		errorValue: fmt.Sprintf("failed to install rosetta: VZErrorDomain Code=%d", vz.ErrorOperationCancelled),
	},
	"TestRosettaNotInstalled-InstallationCancelledButIgnoreIfMissingTrue": {
		installRosetta:  true,
		ignoreIfMissing: true,
		checkRosettaDirectoryShareAvailability: func() vz.LinuxRosettaAvailability {
			return vz.LinuxRosettaAvailabilityNotInstalled
		},
		doInstallRosetta: func() error {
			return fmt.Errorf("VZErrorDomain Code=%d", vz.ErrorOperationCancelled)
		},
	},
	"TestRosettaNotInstalled-InstallationFailedButIgnoreIfMissingTrue": {
		installRosetta:  true,
		ignoreIfMissing: true,
		checkRosettaDirectoryShareAvailability: func() vz.LinuxRosettaAvailability {
			return vz.LinuxRosettaAvailabilityNotInstalled
		},
		doInstallRosetta: func() error {
			return fmt.Errorf("VZErrorDomain Code=%d", vz.ErrorInstallationFailed)
		},
	},
}

func TestCheckRosettaAvailability(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		for name := range checkRosettaAvailabilityTests {
			t.Run(name, func(t *testing.T) {
				test := checkRosettaAvailabilityTests[name]
				testCheckRosettaAvailability(t, &test)
			})
		}
	})
}

func testCheckRosettaAvailability(t *testing.T, test *checkRosettaAvailabilityTest) {
	rosetta :=
		RosettaShare{
			InstallRosetta:  test.installRosetta,
			IgnoreIfMissing: test.ignoreIfMissing,
			DirectorySharingConfig: config.DirectorySharingConfig{
				MountTag: "mount",
			},
		}

	origCheckRosettaDirectoryShareAvailability := checkRosettaDirectoryShareAvailability
	checkRosettaDirectoryShareAvailability = test.checkRosettaDirectoryShareAvailability
	origDoInstallRosetta := doInstallRosetta
	doInstallRosetta = test.doInstallRosetta
	defer func() {
		checkRosettaDirectoryShareAvailability = origCheckRosettaDirectoryShareAvailability
		doInstallRosetta = origDoInstallRosetta
	}()

	err := rosetta.checkRosettaAvailability()

	if test.errorValue != "" {
		require.Error(t, err)
		require.ErrorContains(t, err, test.errorValue)
	} else {
		require.NoError(t, err)
	}
}

func TestRosettaCachingOptions(t *testing.T) {
	tests := map[string]struct {
		socketType config.RosettaCacheSocketType
		socket     string
		errorValue string
	}{
		"UnixSocket": {
			socketType: config.RosettaCacheUnixSocket,
			socket:     "/run/rosettad/rosetta.sock",
		},
		"AbstractSocket": {
			socketType: config.RosettaCacheAbstractSocket,
			socket:     config.DefaultRosettaCacheSocketName,
		},
		"UnixSocketPathTooLong": {
			socketType: config.RosettaCacheUnixSocket,
			socket:     "/" + strings.Repeat("a", 200),
			errorValue: "failed to create rosetta unix socket caching options",
		},
		"InvalidType": {
			socketType: "tcp",
			socket:     "1234",
			errorValue: `invalid rosetta cache socket type "tcp"`,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rosetta := RosettaShare{
				DirectorySharingConfig: config.DirectorySharingConfig{MountTag: "mount"},
				CacheSocketType:        test.socketType,
				CacheSocket:            test.socket,
			}
			options, err := rosetta.cachingOptions()
			if test.errorValue != "" {
				require.ErrorContains(t, err, test.errorValue)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, options)
		})
	}
}
