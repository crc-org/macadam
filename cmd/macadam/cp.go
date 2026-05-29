//go:build amd64 || arm64

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/crc-org/macadam/cmd/macadam/registry"
	provider2 "github.com/crc-org/macadam/pkg/machinedriver/provider"
	"github.com/spf13/cobra"
	pcopy "go.podman.io/podman/v6/pkg/copy"
	"go.podman.io/podman/v6/pkg/machine"
	"go.podman.io/podman/v6/pkg/machine/define"
	"go.podman.io/podman/v6/pkg/machine/shim"
)

var (
	cpRecursive bool

	cpCmd = &cobra.Command{
		Use:               "cp [options] SRC DST",
		Short:             "Copy files to or from a running VM",
		Long:              "Copy files between the host and a running virtual machine. Use 'VMNAME:' prefix for remote paths.",
		Args:              cobra.ExactArgs(2),
		PersistentPreRunE: machinePreRunE,
		RunE:              cpCopy,
		Example: `macadam cp ./file.txt myvm:/tmp/
  macadam cp myvm:/tmp/file.txt ./
  macadam cp -r ./mydir myvm:/tmp/`,
	}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: cpCmd,
	})
	cpCmd.Flags().BoolVarP(&cpRecursive, "recursive", "r", false, "Copy directories recursively")
}

func cpCopy(cmd *cobra.Command, args []string) error {
	src, dst := args[0], args[1]

	srcVM, srcPath, dstVM, dstPath, err := pcopy.ParseSourceAndDestination(src, dst)
	if err != nil {
		return err
	}

	vmName := srcVM
	if vmName == "" {
		vmName = dstVM
	}
	if vmName == "" {
		return errors.New("at least one of SRC or DST must use the VMNAME: prefix")
	}
	if srcVM != "" && dstVM != "" && srcVM != dstVM {
		return fmt.Errorf("SRC and DST reference different VMs (%q and %q)", srcVM, dstVM)
	}

	vmProvider, err := provider2.GetProviderOrDefault(provider)
	if err != nil {
		return err
	}

	mc, _, err := shim.VMExists(vmName)
	if err != nil {
		return err
	}
	if mc == nil {
		return fmt.Errorf("VM %q does not exist", vmName)
	}

	state, err := vmProvider.State(mc, false)
	if err != nil {
		return err
	}
	if state != define.Running {
		return fmt.Errorf("vm %q is not running", mc.Name)
	}

	address := mc.GetAddress()

	src = resolveCpPath(srcPath, srcVM != "", mc.SSH.RemoteUsername, address)
	dst = resolveCpPath(dstPath, dstVM != "", mc.SSH.RemoteUsername, address)

	scpArgs := []string{
		"-i", mc.SSH.IdentityPath,
		"-P", strconv.Itoa(mc.SSH.Port),
	}
	if cpRecursive {
		scpArgs = append(scpArgs, "-r")
	}
	scpArgs = append(scpArgs, machine.LocalhostSSHArgs()...)
	scpArgs = append(scpArgs, src, dst)

	scpExec := exec.CommandContext(cmd.Context(), "scp", scpArgs...)
	scpExec.Stdin = os.Stdin
	scpExec.Stdout = os.Stdout
	scpExec.Stderr = os.Stderr
	return scpExec.Run()
}

func resolveCpPath(path string, isRemote bool, user, address string) string {
	if !isRemote {
		return path
	}
	return fmt.Sprintf("%s@%s:%s", user, address, path)
}
