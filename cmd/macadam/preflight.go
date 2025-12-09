package main

import (
	"runtime"

	"github.com/crc-org/macadam/cmd/macadam/registry"
	provider2 "github.com/crc-org/macadam/pkg/machinedriver/provider"
	"github.com/crc-org/macadam/pkg/preflights"
	"github.com/spf13/cobra"
)

var (
	preflightsCmd = &cobra.Command{
		Use:     "preflight",
		Short:   "Perform preflight checks on an existing machine",
		Long:    "Perform preflight checks on a managed virtual machine ",
		RunE:    preflight,
		Args:    cobra.MaximumNArgs(0),
		Example: `macadam preflight`,
	}

	preflightsOptionalFlags = PreflightsOptionalFlags{}
)

// Flags which have a meaning when unspecified that differs from the flag default
type PreflightsOptionalFlags struct {
	UserModeNetworking bool
}

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: preflightsCmd,
	})

	flags := preflightsCmd.Flags()

	// User-mode networking flag is only available on Windows (HyperV-only)
	if runtime.GOOS == "windows" {
		userModeNetFlagName := "user-mode-networking"
		flags.BoolVar(&preflightsOptionalFlags.UserModeNetworking, userModeNetFlagName, false,
			"Whether this machine should use user-mode networking, routing traffic through a host user-space process (Hyperv-only, requires --provider=hyperv)")
	}
}

func preflight(cmd *cobra.Command, args []string) error {
	vmProvider, err := provider2.GetProviderOrDefault(provider)
	if err != nil {
		return err
	}

	var userModeNetworking *bool
	if cmd.Flags().Changed("user-mode-networking") {
		userModeNetworking = &preflightsOptionalFlags.UserModeNetworking
	}

	return preflights.RunPreflights(vmProvider, userModeNetworking)
}
