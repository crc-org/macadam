//go:build !darwin && !windows

package preflights

import (
	"fmt"

	"github.com/containers/podman/v5/pkg/machine"
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
	return nil
}

func checkKrunKitAvailability(provider vmconfigs.VMProvider) error {
	return nil
}

func getBinariesDirs() []string {
	// On Linux, Podman helper binaries dirs can be found at
	// https://github.com/containers/common/blob/main/pkg/config/config_linux.go#L24-L28
	// We use the same to detect the binaries
	return []string{
		"/usr/local/libexec/podman",
		"/usr/local/lib/podman",
		"/usr/libexec/podman",
		"/usr/lib/podman",
	}
}
