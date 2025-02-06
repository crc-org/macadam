package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cfergeau/macadam/cmd/macadam/registry"
	"github.com/cfergeau/macadam/pkg/cmdline"
	"github.com/cfergeau/macadam/pkg/env"
	"github.com/containers/podman/v5/libpod/define"
	"github.com/spf13/cobra"
)

// HelpTemplate is the help template for podman commands
// This uses the short and long options.
// command should not use this.
const helpTemplate = `{{.Short}}

Description:
  {{.Long}}

{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`

// UsageTemplate is the usage template for podman commands
// This blocks the displaying of the global options. The main podman
// command should not use this.
const usageTemplate = `Usage:{{if (and .Runnable (not .HasAvailableSubCommands))}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.UseLine}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}
{{end}}
`

var (
	rootCmd = &cobra.Command{
		Use:                   filepath.Base(os.Args[0]) + " [options]",
		Long:                  "Manage pods, containers and images",
		SilenceUsage:          true,
		SilenceErrors:         true,
		TraverseChildren:      true,
		Version:               cmdline.Version(),
		DisableFlagsInUseLine: true,
		PersistentPreRunE:     machinePreRunE,
	}

	// defaultLogLevel = "warn"
	// logLevel        = defaultLogLevel
	// dockerConfig    = ""
	// debug           bool

	// requireCleanup = true

	// Defaults for capturing/redirecting the command output since (the) cobra is
	// global-hungry and doesn't allow you to attach anything that allows us to
	// transform the noStdout BoolVar to a string that we can assign to useStdout.
	// noStdout  = false
	// useStdout = ""
)

func init() {
	rootCmd.SetUsageTemplate(usageTemplate)
}

func Execute() {
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		if registry.GetExitCode() == 0 {
			registry.SetExitCode(define.ExecErrorCodeGeneric)
		}
		fmt.Fprintln(os.Stderr, err)
	}

	os.Exit(registry.GetExitCode())
}

func machinePreRunE(c *cobra.Command, args []string) error {
	return env.SetupEnvironment()
}
