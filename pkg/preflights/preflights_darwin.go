//go:build darwin

package preflights

import (
	"fmt"

	"github.com/containers/podman/v5/pkg/machine"
	"github.com/containers/podman/v5/pkg/machine/define"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
)

// macadam/podman needs a gvproxy version which supports the -services
// argument
func checkGvproxyVersion(provider vmconfigs.VMProvider, _ *bool) error {
	if err := checkBinaryArg(machine.ForwarderBinaryName, "-services"); err != nil {
		return fmt.Errorf("%w, please update to gvproxy v0.8.3 or newer", err)
	}
	return nil
}

// macadam/podman needs a vfkit binary which supports the --cloud-init
// argument to inject ssh keys in RHEL cloud images
func checkVfkitVersion(provider vmconfigs.VMProvider) error {
	if provider.VMType() != define.AppleHvVirt {
		return nil
	}
	if err := checkBinaryArg("vfkit", "--cloud-init"); err != nil {
		return fmt.Errorf("%w, please update to vfkit v0.6.1 or newer", err)
	}
	return nil
}

func checkKrunKitAvailability(provider vmconfigs.VMProvider) error {
	if provider.VMType() != define.LibKrun {
		return nil
	}
	if err := checkBinaryArg("krunkit", "--version"); err != nil {
		return fmt.Errorf("%w, please install krunkit", err)
	}
	return nil
}

func getBinariesDirs() []string {
	// On Mac, Podman helper binaries dirs can be found at
	// https://github.com/containers/common/blob/main/pkg/config/config_darwin.go#L15-L28
	// We use the same to detect the binaries
	return []string{
		// Relative to the binary directory
		"$BINDIR/../libexec/podman",
		// Homebrew install paths
		"/usr/local/opt/podman/libexec/podman",
		"/opt/homebrew/opt/podman/libexec/podman",
		"/opt/homebrew/bin",
		"/usr/local/bin",
		// default paths
		"/usr/local/libexec/podman",
		"/usr/local/lib/podman",
		"/usr/libexec/podman",
		"/usr/lib/podman",
	}
}
