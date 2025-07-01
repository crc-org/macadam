//go:build amd64 || arm64

package main

import (
	"fmt"

	"github.com/containers/podman/v5/pkg/machine"
	"github.com/containers/podman/v5/pkg/machine/shim"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
	"github.com/crc-org/macadam/cmd/macadam/registry"
	macadam "github.com/crc-org/macadam/pkg/machinedriver"
	provider2 "github.com/crc-org/macadam/pkg/machinedriver/provider"
	"github.com/spf13/cobra"
)

var (
	startCmd = &cobra.Command{
		Use:     "start [options] [MACHINE]",
		Short:   "Start an existing machine",
		Long:    "Start a managed virtual machine",
		RunE:    start,
		Args:    cobra.MaximumNArgs(1),
		Example: `macadam start`,
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
	if len(args) > 0 && len(args[0]) > 0 {
		machineName = args[0]
	}
	initOpts := macadam.DefaultInitOpts(machineName)
	//initOpts.ImagePuller = ...
	vmProvider, err := provider2.GetProviderOrDefault(provider)
	if err != nil {
		return err
	}
	vmConfig, _, err := shim.VMExists(initOpts.Name, []vmconfigs.VMProvider{vmProvider})
	if err != nil {
		return err
	}
	if vmConfig == nil {
		return fmt.Errorf("VM %s does not exist", machineName)
	}

	return macadam.Start(vmConfig, vmProvider)
}
