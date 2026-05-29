//go:build amd64 || arm64

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/crc-org/macadam/cmd/macadam/registry"
	provider2 "github.com/crc-org/macadam/pkg/machinedriver/provider"
	"github.com/spf13/cobra"
	"go.podman.io/podman/v6/pkg/machine"
	"go.podman.io/podman/v6/pkg/machine/define"
	"go.podman.io/podman/v6/pkg/machine/shim"
)

var (
	portForwardAddress string

	portForwardCmd = &cobra.Command{
		Use:               "port-forward [options] NAME [LOCAL_PORT:]REMOTE_PORT [...[LOCAL_PORT:]REMOTE_PORT]",
		Aliases:           []string{"pf"},
		Short:             "Forward one or more local ports to a running VM",
		Long:              "Forward one or more local ports to a running virtual machine over SSH",
		Args:              cobra.MinimumNArgs(2),
		PersistentPreRunE: machinePreRunE,
		RunE:              runPortForward,
		Example: `macadam port-forward myvm 8080:8080
  macadam port-forward myvm 8080
  macadam port-forward myvm 8080:8080 9090:9090
  macadam port-forward --address 0.0.0.0 myvm 8080:8080`,
	}
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: portForwardCmd,
	})
	portForwardCmd.Flags().StringVar(&portForwardAddress, "address", "localhost", "Local address to bind (e.g. 0.0.0.0 for all interfaces)")
}

func runPortForward(cmd *cobra.Command, args []string) error {
	vmName, portSpecs := args[0], args[1:]

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

	sshArgs := []string{
		"-i", mc.SSH.IdentityPath,
		"-p", strconv.Itoa(mc.SSH.Port),
	}
	for _, spec := range portSpecs {
		lFlag, err := toSSHForwardFlag(spec, portForwardAddress)
		if err != nil {
			return err
		}
		sshArgs = append(sshArgs, "-L", lFlag)
	}
	sshArgs = append(sshArgs, machine.LocalhostSSHArgs()...)
	sshArgs = append(sshArgs, "-N", mc.SSH.RemoteUsername+"@"+address)

	for _, spec := range portSpecs {
		fmt.Fprintf(os.Stderr, "Forwarding from %s -> %s\n", spec, vmName)
	}
	fmt.Fprintln(os.Stderr, "Press Ctrl+C to stop forwarding.")

	sshExec := exec.CommandContext(cmd.Context(), "ssh", sshArgs...)
	sshExec.Stdin = os.Stdin
	sshExec.Stdout = os.Stdout
	sshExec.Stderr = os.Stderr
	return sshExec.Run()
}

// toSSHForwardFlag converts "[LOCAL:]REMOTE" into the SSH -L format:
// "BIND_ADDR:LOCAL_PORT:127.0.0.1:REMOTE_PORT"
func toSSHForwardFlag(spec, bindAddr string) (string, error) {
	parts := strings.SplitN(spec, ":", 2)
	localPort := parts[0]
	remotePort := parts[0]
	if len(parts) == 2 {
		remotePort = parts[1]
	}
	if err := validatePortNumber(localPort, spec); err != nil {
		return "", err
	}
	if err := validatePortNumber(remotePort, spec); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s:127.0.0.1:%s", bindAddr, localPort, remotePort), nil
}

func validatePortNumber(port, spec string) error {
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("invalid port spec %q: %q must be a number between 1 and 65535", spec, port)
	}
	return nil
}
