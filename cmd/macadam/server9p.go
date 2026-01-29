//go:build windows && (amd64 || arm64)

// Derived from Podman's server9p implementation.
// Source: https://github.com/containers/podman/blob/v5.7/cmd/podman/machine/server9p.go
//
// Integrated as a subcommand of 'machine' to ensure compatibility with Podman
// The main difference is that we replaced the original shutdown logic with
// signal-based handling to ensure a graceful exit.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/containers/common/pkg/completion"
	"github.com/containers/podman/v5/pkg/fileserver"
	"github.com/crc-org/macadam/cmd/macadam/registry"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var server9pCommand = &cobra.Command{
	Args:              cobra.NoArgs,
	Use:               "server9p [options]",
	Hidden:            false,
	Short:             "Serve a directory using 9p over hvsock",
	Long:              "Start a number of 9p servers on given hvsock UUIDs. Run until interrupted by signal (SIGTERM or SIGINT).",
	RunE:              remoteDirServer,
	ValidArgsFunction: completion.AutocompleteNone,
	Example:           `macadam machine server9p --serve C:\Users\myuser:00000050-FACB-11E6-BD58-64006A7986D3`,
}

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: server9pCommand,
		Parent:  machineCmd,
	})

	flags := server9pCommand.Flags()

	serveFlagName := "serve"
	flags.StringArrayVar(&serveDirs, serveFlagName, []string{}, "directories to serve and UUID of vsock to serve on, colon-separated")
	_ = server9pCommand.RegisterFlagCompletionFunc(serveFlagName, completion.AutocompleteNone)
}

var serveDirs []string

func remoteDirServer(_ *cobra.Command, args []string) error {
	if len(serveDirs) == 0 {
		return fmt.Errorf("must provide at least one directory to serve")
	}

	// TODO: need to support options here
	shares := make(map[string]string, len(serveDirs))
	for _, share := range serveDirs {
		splitShare := strings.Split(share, ":")
		if len(splitShare) < 2 {
			return fmt.Errorf("paths passed to --serve must include an hvsock GUID")
		}

		// Every element but the last one is the real filepath to share
		path := strings.Join(splitShare[:len(splitShare)-1], ":")

		shares[path] = splitShare[len(splitShare)-1]
	}

	if err := fileserver.StartShares(shares); err != nil {
		return err
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	logrus.Info("9p server started, waiting for signal to shutdown...")
	sig := <-sigChan
	logrus.Infof("Received signal %v, shutting down 9p server", sig)

	return nil
}
