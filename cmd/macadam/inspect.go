//go:build amd64 || arm64

package main

import (
	"encoding/json"
	"os"
	"time"

	"github.com/containers/podman/v5/cmd/podman/utils"
	"github.com/containers/podman/v5/pkg/machine/define"
	"github.com/containers/podman/v5/pkg/machine/env"
	providerpkg "github.com/containers/podman/v5/pkg/machine/provider"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
	"github.com/crc-org/macadam/cmd/macadam/registry"
	"github.com/spf13/cobra"
)

var (
	inspectCmd = &cobra.Command{
		Use:               "inspect [options] [MACHINE...]",
		Short:             "Inspect an existing machine",
		Long:              "Provide details on a managed virtual machine",
		PersistentPreRunE: machinePreRunE,
		RunE:              inspect,
		Example:           `podman machine inspect myvm`,
		//ValidArgsFunction: autocompleteMachine,
	}
)

// this is based on the struct of the same name in
// github.com/containers/podman/v5/pkg/machine/config.go
type InspectInfo struct {
	ConfigDir          define.VMFile
	Created            time.Time
	LastUp             *time.Time `json:",omitempty"`
	Name               string
	Resources          vmconfigs.ResourceConfig
	SSHConfig          vmconfigs.SSHConfig
	State              define.Status
	UserModeNetworking bool
}

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: inspectCmd,
	})
}

func inspect(cmd *cobra.Command, args []string) error {
	var (
		errs utils.OutputErrors
	)
	provider, err := providerpkg.Get()
	if err != nil {
		return nil
	}
	dirs, err := env.GetMachineDirs(provider.VMType())
	if err != nil {
		return err
	}
	if len(args) < 1 {
		args = append(args, defaultMachineName)
	}

	vms := make([]InspectInfo, 0, len(args))
	for _, name := range args {
		mc, err := vmconfigs.LoadMachineByName(name, dirs)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		state, err := provider.State(mc, false)
		if err != nil {
			return err
		}

		ii := InspectInfo{
			ConfigDir:          *dirs.ConfigDir,
			Created:            mc.Created,
			LastUp:             &mc.LastUp,
			Name:               mc.Name,
			Resources:          mc.Resources,
			SSHConfig:          mc.SSH,
			State:              state,
			UserModeNetworking: provider.UserModeNetworkEnabled(mc),
		}
		if ii.LastUp.IsZero() {
			ii.LastUp = nil
		}

		vms = append(vms, ii)
	}

	if err := printJSON(vms); err != nil {
		errs = append(errs, err)
	}
	return errs.PrintErrors()
}

func printJSON(data []InspectInfo) error {
	enc := json.NewEncoder(os.Stdout)
	// by default, json marshallers will force utf=8 from
	// a string. this breaks healthchecks that use <,>, &&.
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "     ")
	return enc.Encode(data)
}
