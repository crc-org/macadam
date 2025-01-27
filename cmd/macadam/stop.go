package main

import (
	"github.com/cfergeau/macadam/cmd/macadam/registry"
	macadam "github.com/cfergeau/macadam/pkg/machinedriver"
	"github.com/spf13/cobra"
)

var (
	stopCmd = &cobra.Command{
		Use:     "stop",
		Short:   "Stop an existing machine",
		Long:    "Stop a managed virtual machine ",
		RunE:    stop,
		Args:    cobra.MaximumNArgs(0),
		Example: `macadam stop`,
	}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: stopCmd,
	})
}

func stop(cmd *cobra.Command, args []string) error {
	driver, err := macadam.GetDriverByMachineName(defaultMachineName)
	if err != nil {
		return nil
	}

	return driver.Stop()
}
