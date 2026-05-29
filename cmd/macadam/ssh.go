//go:build amd64 || arm64

package main

import (
	"errors"
	"fmt"

	"go.podman.io/podman/v6/pkg/machine/define"
	"go.podman.io/podman/v6/pkg/machine/shim"

	"github.com/crc-org/macadam/cmd/macadam/registry"
	provider2 "github.com/crc-org/macadam/pkg/machinedriver/provider"
	"github.com/spf13/cobra"
	"go.podman.io/common/pkg/completion"
	"go.podman.io/podman/v6/cmd/podman/utils"
	"go.podman.io/podman/v6/pkg/machine"
	"go.podman.io/podman/v6/pkg/machine/vmconfigs"
)

var (
	sshCmd = &cobra.Command{
		Use:               "ssh [options] [NAME] [COMMAND [ARG ...]]",
		Short:             "SSH into an existing machine",
		Long:              "SSH into a managed virtual machine ",
		PersistentPreRunE: machinePreRunE,
		RunE:              ssh,
		Example: `macadam ssh podman-machine-default
  macadam ssh myvm echo hello`,
		//ValidArgsFunction: autocompleteMachineSSH,
	}
)

var (
	sshOpts machine.SSHOptions
)

func init() {
	sshCmd.Flags().SetInterspersed(false)
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: sshCmd,
	})
	flags := sshCmd.Flags()
	usernameFlagName := "username"
	flags.StringVar(&sshOpts.Username, usernameFlagName, "", "Username to use when ssh-ing into the VM.")
	_ = sshCmd.RegisterFlagCompletionFunc(usernameFlagName, completion.AutocompleteNone)
}

// TODO Remember that this changed upstream and needs to updated as such!

func ssh(cmd *cobra.Command, args []string) error {
	var (
		err error
		mc  *vmconfigs.MachineConfig
	)

	vmProvider, err := provider2.GetProviderOrDefault(provider)
	if err != nil {
		return err
	}

	// Set the VM to default
	vmName := defaultMachineName
	// If len is greater than 0, it means we may have been
	// provided the VM name.  If so, we check.  The VM name,
	// if provided, must be in args[0].
	if len(args) > 0 {
		// note: previous incantations of this up by a specific name
		// and errors were ignored.  this error is not ignored because
		// it implies podman cannot read its machine files, which is bad
		mc, _, err = shim.VMExists(args[0])
		if err != nil {
			return err
		}

		if mc != nil {
			vmName = args[0]
		} else {
			sshOpts.Args = append(sshOpts.Args, args[0])
		}
	}

	// If len is greater than 1, it means we might have been
	// given a vmname and args or just args
	if len(args) > 1 {
		if mc != nil {
			sshOpts.Args = args[1:]
		} else {
			sshOpts.Args = args
		}
	}

	// If the machine config was not loaded earlier, we load it now
	if mc == nil {
		mc, _, err = shim.VMExists(vmName)
		// we just return generic error message as we cannot be sure the vm should have the default name or not
		// in the previous if branch we tested with args[0] and it did not exist
		// here we tried with the default name.
		// If we fail here it could mean the machine has been deleted externally (e.g through hyperv manager) and
		// we cannot be sure what the user is targetting.
		if err != nil {
			return fmt.Errorf("VM not found: %w", err)
		}
		if mc == nil {
			return errors.New("VM does not exist")
		}
	}

	state, err := vmProvider.State(mc, false)
	if err != nil {
		return err
	}
	if state != define.Running {
		return fmt.Errorf("vm %q is not running", mc.Name)
	}

	username := sshOpts.Username
	if username == "" {
		username = mc.SSH.RemoteUsername
	}

	err = machine.LocalhostSSHShellWithAddress(username, mc.SSH.IdentityPath, mc.Name, mc.GetAddress(), mc.SSH.Port, sshOpts.Args)
	return utils.HandleOSExecError(err)
}
