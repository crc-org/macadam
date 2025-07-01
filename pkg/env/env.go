package env

import (
	"os"
	"path/filepath"

	"github.com/containers/podman/v5/pkg/machine/define"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
	"github.com/containers/storage/pkg/homedir"
)

const connectionsFile = "macadam-connections.json"

func SetupEnvironment(provider vmconfigs.VMProvider) error {
	path, err := homedir.GetConfigHome()
	if err != nil {
		return err
	}

	connsFile := filepath.Join(filepath.Dir(path), "macadam", connectionsFile)
	// set the path used for storing connection of macadam vms
	err = os.Setenv("PODMAN_CONNECTIONS_CONF", connsFile)
	if err != nil {
		return err
	}

	// set the directory used when calculating the data and config paths
	// config -> <configHome>/containers/macadam/machine (configHome changes based on the OS used e.g. configHome == /home/user/.config)
	// data -> <dataHome>/containers/macadam/machine (dataHome changes based on the OS used e.g. dataHome == /home/user/.local/share)
	err = os.Setenv("PODMAN_DATA_DIR", filepath.Join("macadam", "machine"))
	if err != nil {
		return err
	}

	// set the directory to be used when calculating runtime path
	// run -> <runHome>/macadam (runHome changes based on the OS used e.g. runHome == /run)
	err = os.Setenv("PODMAN_RUNTIME_DIR", "macadam")
	if err != nil {
		return err
	}

	// set the prefix that will be used when creating the wsl distro
	// if this is not set, every dist will have "podman" as prefix
	if provider.VMType() == define.WSLVirt {
		err = os.Setenv("PODMAN_TOOL_PREFIX", "macadam")
		if err != nil {
			return err
		}
	}

	return nil
}
