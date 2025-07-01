package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/common/pkg/completion"
	"github.com/containers/podman/v5/libpod/define"
	"github.com/crc-org/macadam/cmd/macadam/common"
	"github.com/crc-org/macadam/cmd/macadam/registry"
	"github.com/crc-org/macadam/pkg/cmdline"
	"github.com/crc-org/macadam/pkg/env"
	provider2 "github.com/crc-org/macadam/pkg/machinedriver/provider"
	"github.com/sirupsen/logrus"
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

	defaultLogLevel = "warn"
	logLevel        = defaultLogLevel
	provider        = ""
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
	// Hooks are called before PersistentPreRunE(). These hooks affect global
	// state and are executed after processing the command-line, but before
	// actually running the command.
	cobra.OnInitialize(
		loggingHook,
	)

	pFlags := rootCmd.PersistentFlags()

	logLevelFlagName := "log-level"
	pFlags.StringVar(&logLevel, logLevelFlagName, logLevel, fmt.Sprintf("Log messages above specified level (%s)", strings.Join(common.LogLevels, ", ")))
	_ = rootCmd.RegisterFlagCompletionFunc(logLevelFlagName, common.AutocompleteLogLevel)

	providerFlagName := "provider"
	pFlags.StringVar(&provider, providerFlagName, "", fmt.Sprintf("Name for the provider (%s). Default value: %s", strings.Join(provider2.GetProviders(), ", "), provider2.GetDefaultProvider()))
	_ = initCmd.RegisterFlagCompletionFunc(providerFlagName, completion.AutocompleteNone)

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
	vmProvider, err := provider2.GetProviderOrDefault(provider)
	if err != nil {
		return err
	}

	return env.SetupEnvironment(vmProvider)
}

func loggingHook() {
	var found bool
	for _, l := range common.LogLevels {
		if l == strings.ToLower(logLevel) {
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintf(os.Stderr, "Log Level %q is not supported, choose from: %s\n", logLevel, strings.Join(common.LogLevels, ", "))
		os.Exit(1)
	}

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
	logrus.SetLevel(level)

	if logrus.IsLevelEnabled(logrus.InfoLevel) {
		logrus.Infof("%s filtering at log level %s", os.Args[0], logrus.GetLevel())
	}
}
