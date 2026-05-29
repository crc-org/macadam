//go:build amd64 || arm64

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/crc-org/macadam/cmd/macadam/registry"
	provider2 "github.com/crc-org/macadam/pkg/machinedriver/provider"
	"github.com/crc-org/macadam/pkg/preflights"
	"github.com/spf13/cobra"
	"go.podman.io/podman/v6/cmd/podman/utils"
	"go.podman.io/podman/v6/pkg/machine"
	"go.podman.io/podman/v6/pkg/machine/define"
	"go.podman.io/podman/v6/pkg/machine/env"
	"go.podman.io/podman/v6/pkg/machine/shim"
	"go.podman.io/podman/v6/pkg/machine/vmconfigs"
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

type GVProxyInfo struct {
	Binary        *define.VMFile `json:",omitempty"`
	ServiceSocket *define.VMFile `json:",omitempty"`
	Logs          *define.VMFile `json:",omitempty"`
}

// HostServicesInfo contains information about host-side services and processes
// that support the VM, such as gvproxy
type HostServicesInfo struct {
	GVProxy *GVProxyInfo `json:",omitempty"`
}

// this is based on the struct of the same name in
// go.podman.io/podman/v6/pkg/machine/config.go
type InspectInfo struct {
	ConfigDir          define.VMFile
	Created            time.Time
	LastUp             *time.Time `json:",omitempty"`
	Name               string
	Resources          vmconfigs.ResourceConfig
	SSHConfig          vmconfigs.SSHConfig
	State              define.Status
	UserModeNetworking bool
	Services           *HostServicesInfo `json:",omitempty"`
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
	vmProvider, err := provider2.GetProviderOrDefault(provider)
	if err != nil {
		return err
	}
	dirs, err := env.GetMachineDirs(vmProvider.VMType())
	if err != nil {
		return err
	}
	if len(args) < 1 {
		args = append(args, defaultMachineName)
	}

	vms := make([]InspectInfo, 0, len(args))
	for _, name := range args {
		mc, _, err := shim.VMExists(name)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if mc == nil {
			errs = append(errs, fmt.Errorf("VM %q does not exist", name))
			continue
		}

		state, err := vmProvider.State(mc, false)
		if err != nil {
			return err
		}

		// Gather service information
		// gvproxy is used when UserModeNetworkEnabled returns true
		var servicesInfo *HostServicesInfo
		if vmProvider.UserModeNetworkEnabled(mc) {
			servicesInfo = &HostServicesInfo{
				GVProxy: &GVProxyInfo{},
			}
			if binary, err := preflights.GetBinaryPath(machine.ForwarderBinaryName); err == nil {
				servicesInfo.GVProxy.Binary = binary
			}
			if gvproxyServiceSocket, err := mc.GVProxyServiceSocket(); err == nil {
				servicesInfo.GVProxy.ServiceSocket = gvproxyServiceSocket
			}
			if gvproxyLogFile, err := machine.GetGVProxyLogFile(mc, dirs); err == nil {
				servicesInfo.GVProxy.Logs = gvproxyLogFile
			}
		}

		ii := InspectInfo{
			ConfigDir:          *dirs.ConfigDir,
			Created:            mc.Created,
			LastUp:             &mc.LastUp,
			Name:               mc.Name,
			Resources:          mc.Resources,
			SSHConfig:          mc.SSH,
			State:              state,
			UserModeNetworking: vmProvider.UserModeNetworkEnabled(mc),
			Services:           servicesInfo,
		}
		if ii.LastUp.IsZero() {
			ii.LastUp = nil
		}

		vms = append(vms, ii)
	}

	// Only print JSON if we have at least one successful VM
	// This prevents printing an empty [] when all machines failed
	if len(vms) > 0 {
		if err := printJSON(vms); err != nil {
			errs = append(errs, err)
		}
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
