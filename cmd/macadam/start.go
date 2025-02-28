//go:build amd64 || arm64

package main

import (
	"fmt"

	"github.com/crc-org/macadam/cmd/macadam/registry"
	macadam "github.com/crc-org/macadam/pkg/machinedriver"
	ldefine "github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/machine"
	"github.com/containers/podman/v5/pkg/machine/provider"
	"github.com/containers/podman/v5/pkg/machine/shim"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
	"github.com/spf13/cobra"
)

var (
	startCmd = &cobra.Command{
		Use:     "start [options] [MACHINE]",
		Short:   "Start an existing machine",
		Long:    "Start a managed virtual machine ",
		RunE:    start,
		Args:    cobra.MaximumNArgs(1),
		Example: `macadam start podman-machine-default`,
	}
	startOpts = machine.StartOptions{}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: startCmd,
	})

	flags := startCmd.Flags()
	noInfoFlagName := "no-info"
	flags.BoolVar(&startOpts.NoInfo, noInfoFlagName, false, "Suppress informational tips")

	quietFlagName := "quiet"
	flags.BoolVarP(&startOpts.Quiet, quietFlagName, "q", false, "Suppress machine starting status output")
}

func start(_ *cobra.Command, args []string) error {
	machineName := defaultMachineName
	if len(args) > 0 {
		if len(args[0]) > maxMachineNameSize {
			return fmt.Errorf("machine name %q must be %d characters or less", args[0], maxMachineNameSize)
		}
		machineName = args[0]

		if !ldefine.NameRegex.MatchString(machineName) {
			return fmt.Errorf("invalid name %q: %w", machineName, ldefine.RegexError)
		}
	}
	initOpts := macadam.DefaultInitOpts(machineName)
	//initOpts.ImagePuller = ...
	vmProvider, err := provider.Get()
	if err != nil {
		return nil
	}
	vmConfig, _, err := shim.VMExists(initOpts.Name, []vmconfigs.VMProvider{vmProvider})
	if err != nil {
		return err
	}
	if vmConfig == nil {
		err = shim.Init(*initOpts, vmProvider)
		if err != nil {
			return err
		}
	}

	return macadam.Start(vmConfig, vmProvider)
}
