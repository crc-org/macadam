package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/containers/common/pkg/completion"
	"github.com/containers/common/pkg/report"
	"github.com/containers/podman/v5/pkg/domain/entities"
	provider2 "github.com/containers/podman/v5/pkg/machine/provider"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
	"github.com/crc-org/macadam/cmd/macadam/registry"
	macadam "github.com/crc-org/macadam/pkg/machinedriver"
	"github.com/crc-org/machine/libmachine/state"
	"github.com/docker/go-units"
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
	format    string
	noHeading bool
	quiet     bool
}

type ListReporter struct {
	Name           string
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
	flags.BoolVarP(&listFlag.noHeading, "noheading", "n", false, "Do not print headers")
	flags.BoolVarP(&listFlag.quiet, "quiet", "q", false, "Show only machine names")
}

func list(cmd *cobra.Command, args []string) error {
	provider, err := provider2.Get()
	if err != nil {
		return err
	}
	providers := []vmconfigs.VMProvider{provider}

	listDrivers, err := macadam.List(providers)
	if err != nil {
		return err
	}

	if report.IsJSON(listFlag.format) {
		machineReporter := toMachineFormat(listDrivers)
		b, err := json.MarshalIndent(machineReporter, "", "    ")
		if err != nil {
			return err
		}
		os.Stdout.Write(b)

		return nil
	}
	machineReporter := toHumanFormat(listDrivers)
	return outputTemplate(cmd, machineReporter)
}

func outputTemplate(cmd *cobra.Command, responses []ListReporter) error {
	headers := report.Headers(entities.ListReporter{}, map[string]string{
		"LastUp":   "LAST UP",
		"VmType":   "VM TYPE",
		"CPUs":     "CPUS",
		"Memory":   "MEMORY",
		"DiskSize": "DISK SIZE",
	})

	rpt := report.New(os.Stdout, cmd.Name())
	defer rpt.Flush()

	var err error
	switch {
	case cmd.Flag("format").Changed:
		rpt, err = rpt.Parse(report.OriginUser, listFlag.format)
	case listFlag.quiet:
		rpt, err = rpt.Parse(report.OriginUser, "{{.Name}}\n")
	default:
		rpt, err = rpt.Parse(report.OriginPodman, listFlag.format)
	}
	if err != nil {
		return err
	}

	if rpt.RenderHeaders && !listFlag.noHeading {
		if err := rpt.Execute(headers); err != nil {
			return fmt.Errorf("failed to write report column headers: %w", err)
		}
	}
	return rpt.Execute(responses)
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

func toMachineFormat(drivers []*macadam.Driver) []ListReporter {
	machineResponses := []ListReporter{}

	for _, d := range drivers {
		vm := d.GetVmConfig()

		vmState, err := d.GetState()
		if err != nil {
			return machineResponses
		}

		response := new(ListReporter)
		response.Name = vm.Name
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
	}

	return machineResponses
}

func toHumanFormat(drivers []*macadam.Driver) []ListReporter {
	humanResponses := []ListReporter{}

	for _, d := range drivers {
		vm := d.GetVmConfig()

		vmState, err := d.GetState()
		if err != nil {
			return humanResponses
		}

		response := new(ListReporter)
		response.Name = vm.Name
		response.LastUp = strTime(vm.LastUp)
		switch {
		case vm.Starting:
			response.LastUp = "Currently starting"
			response.Starting = true
		case vmState == state.Running:
			response.LastUp = "Currently running"
			response.Running = true
		case vm.LastUp.IsZero():
			response.LastUp = "Never"
		default:
			response.LastUp = units.HumanDuration(time.Since(vm.LastUp)) + " ago"
		}
		response.Created = units.HumanDuration(time.Since(vm.Created)) + " ago"
		response.CPUs = vm.Resources.CPUs
		response.Memory = units.BytesSize(float64(vm.Resources.Memory.ToBytes()))
		response.DiskSize = units.BytesSize(float64(vm.Resources.DiskSize.ToBytes()))
		response.Port = vm.SSH.Port
		response.RemoteUsername = vm.SSH.RemoteUsername
		response.IdentityPath = vm.SSH.IdentityPath
		response.VMType = d.GetVMType().String()

		humanResponses = append(humanResponses, *response)
	}
	return humanResponses
}
