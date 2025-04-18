package main

import (
	"github.com/crc-org/macadam/cmd/macadam/registry"
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
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: preflightsCmd,
	})
}

func preflight(_ *cobra.Command, args []string) error {
	return preflights.RunPreflights()
}
