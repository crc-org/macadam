package main

import (
	"github.com/crc-org/macadam/cmd/macadam/registry"
	"github.com/spf13/cobra"
)

var (
	machineCmd = &cobra.Command{
		Use:   "machine",
		Short: "Manage virtual machines",
		Long:  "Manage virtual machines",
	}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: machineCmd,
	})
}
