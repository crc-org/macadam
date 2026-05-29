package system

import (
	"github.com/crc-org/macadam/cmd/macadam/registry"
	"github.com/spf13/cobra"
	"go.podman.io/podman/v6/cmd/podman/validate"
)

var (
	// Command: macadam _system_
	systemCmd = &cobra.Command{
		Use:   "system",
		Short: "Manage Macadam",
		Long:  "Manage Macadam",
		RunE:  validate.SubCommandExists,
	}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: systemCmd,
	})
}
