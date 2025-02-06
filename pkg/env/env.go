package env

import (
	"os"
	"path/filepath"

	"github.com/containers/storage/pkg/homedir"
)

const connectionsFile = "macadam-connections.json"

func SetupEnvironment() error {
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

	return nil
}
