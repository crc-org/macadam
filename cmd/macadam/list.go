package main

import (
	"encoding/json"
	"os"
	"strconv"
	"time"

	"github.com/cfergeau/macadam/cmd/macadam/registry"
	macadam "github.com/cfergeau/macadam/pkg/machinedriver"
	"github.com/containers/common/pkg/completion"
	"github.com/crc-org/machine/libmachine/state"
	"github.com/spf13/cobra"
)

var (
	lsCmd = &cobra.Command{
		Use:     "list [options]",
		Aliases: []string{"ls"},
		Short:   "List machines",
		Long:    "List managed virtual machines.",
		// do not use machinePreRunE, as that pre-sets the provider
		RunE:              list,
		Args:              cobra.MaximumNArgs(0),
		ValidArgsFunction: completion.AutocompleteNone,
		Example: `macadam list,
  macadam list --format json
  macadam ls`,
	}
	listFlag = listFlagType{}
)

type listFlagType struct {
	format string
}

type ListReporter struct {
	Image          string
	Created        string
	Running        bool
	Starting       bool
	LastUp         string
	CPUs           uint64
	Memory         string
	DiskSize       string
	Port           int
	RemoteUsername string
	IdentityPath   string
	VMType         string
}

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: lsCmd,
	})

	flags := lsCmd.Flags()
	formatFlagName := "format"
	flags.StringVar(&listFlag.format, formatFlagName, "{{range .}}{{.Name}}\t{{.VMType}}\t{{.Created}}\t{{.LastUp}}\t{{.CPUs}}\t{{.Memory}}\t{{.DiskSize}}\n{{end -}}", "Format volume output using JSON or a Go template")
}

func list(cmd *cobra.Command, args []string) error {
	driver, err := macadam.GetDriverByMachineName(defaultMachineName)
	if err != nil {
		return nil
	}

	machineReporter := toMachineFormat(driver)
	b, err := json.MarshalIndent(machineReporter, "", "    ")
	if err != nil {
		return err
	}
	os.Stdout.Write(b)
	return nil

}

func strTime(t time.Time) string {
	iso, err := t.MarshalText()
	if err != nil {
		return ""
	}
	return string(iso)
}

func strUint(u uint64) string {
	return strconv.FormatUint(u, 10)
}

func toMachineFormat(d *macadam.Driver) []ListReporter {
	machineResponses := make([]ListReporter, 0, 1)

	vm := d.GetVmConfig()

	vmState, err := d.GetState()
	if err != nil {
		return machineResponses
	}

	response := new(ListReporter)
	response.Image = vm.ImagePath.Path
	response.Running = vmState == state.Running
	response.LastUp = strTime(vm.LastUp)
	response.Created = strTime(vm.Created)
	response.CPUs = vm.Resources.CPUs
	response.Memory = strUint(uint64(vm.Resources.Memory.ToBytes()))
	response.DiskSize = strUint(uint64(vm.Resources.DiskSize.ToBytes()))
	response.Port = vm.SSH.Port
	response.RemoteUsername = vm.SSH.RemoteUsername
	response.IdentityPath = vm.SSH.IdentityPath
	response.Starting = vm.Starting
	response.VMType = d.GetVMType().String()

	machineResponses = append(machineResponses, *response)

	return machineResponses
}
