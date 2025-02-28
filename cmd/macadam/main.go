package main

import (
	"fmt"
	"os"

	"github.com/crc-org/macadam/cmd/macadam/registry"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd = parseCommands()

	Execute()
	os.Exit(0)
}

func parseCommands() *cobra.Command {
	for _, c := range registry.Commands {
		addCommand(c)
	}

	rootCmd.SetFlagErrorFunc(flagErrorFuncfunc)
	return rootCmd
}

func flagErrorFuncfunc(c *cobra.Command, e error) error {
	e = fmt.Errorf("%w\nSee '%s --help'", e, c.CommandPath())
	return e
}

func addCommand(c registry.CliCommand) {
	parent := rootCmd
	if c.Parent != nil {
		parent = c.Parent
	}
	parent.AddCommand(c.Command)

	c.Command.SetFlagErrorFunc(flagErrorFuncfunc)

	// - templates need to be set here, as PersistentPreRunE() is
	// not called when --help is used.
	// - rootCmd uses cobra default template not ours
	c.Command.SetHelpTemplate(helpTemplate)
	c.Command.SetUsageTemplate(usageTemplate)
	c.Command.DisableFlagsInUseLine = true
}
