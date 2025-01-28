package main

import (
	"github.com/cfergeau/macadam/cmd/macadam/registry"
	macadam "github.com/cfergeau/macadam/pkg/machinedriver"
	"github.com/containers/podman/v5/pkg/machine"
	"github.com/spf13/cobra"
)

var (
	rmCmd = &cobra.Command{
		Use:     "rm [options]",
		Short:   "Remove an existing machine",
		Long:    "Remove a managed virtual machine ",
		RunE:    rm,
		Args:    cobra.MaximumNArgs(0),
		Example: `macadam rm`,
	}
)

var (
	destroyOptions machine.RemoveOptions
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: rmCmd,
	})
}

func rm(_ *cobra.Command, args []string) error {
	driver, err := macadam.GetDriverByMachineName(defaultMachineName)
	if err != nil {
		return nil
	}

	return driver.Remove()
}
