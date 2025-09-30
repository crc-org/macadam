//go:build windows

package preflights

import (
	"fmt"

	"github.com/containers/podman/v5/pkg/machine"
	"github.com/containers/podman/v5/pkg/machine/define"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
)

// macadam/podman needs a gvproxy version which supports the --services
// argument
func checkGvproxyVersion(provider vmconfigs.VMProvider, userModeNetworking *bool) error {
	if provider.VMType() == define.WSLVirt || (provider.VMType() == define.HyperVVirt && (userModeNetworking == nil || !*userModeNetworking)) {
		return nil
	}
	if err := checkBinaryArg(machine.ForwarderBinaryName, "-services"); err != nil {
		return fmt.Errorf("%w, please update to gvproxy v0.8.3 or newer", err)
	}
	return nil
}

func checkVfkitVersion(provider vmconfigs.VMProvider) error {
	return nil
}

func checkKrunKitAvailability(provider vmconfigs.VMProvider) error {
	return nil
}

func getBinariesDirs() []string {
	// On Windows, Podman helper binaries dir is "C:\Program Files\RedHat\Podman"
	// https://github.com/containers/common/blob/main/pkg/config/config_windows.go#L33-L36
	// We use the same to detect the binaries
	return []string{
		"C:\\Program Files\\RedHat\\Podman",
	}
}
