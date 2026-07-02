//go:build amd64 || arm64

package main

import (
	"fmt"
	"runtime"

	"github.com/containers/podman/v5/pkg/machine/define"
	"github.com/containers/podman/v5/pkg/machine/shim"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
	"github.com/crc-org/macadam/cmd/macadam/registry"
	provider2 "github.com/crc-org/macadam/pkg/machinedriver/provider"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/spf13/cobra"
	"go.podman.io/common/pkg/completion"
	"go.podman.io/common/pkg/strongunits"
)

var (
	setCmd = &cobra.Command{
		Use:               "set [options] [MACHINE]",
		Short:             "Set a virtual machine setting",
		Long:              "Set an updatable virtual machine setting",
		RunE:              setMachine,
		Args:              cobra.MaximumNArgs(1),
		Example:           `macadam set --cpus 4 --memory 8192 --disk-size 30`,
		ValidArgsFunction: completion.AutocompleteNone,
	}

	setOptsFromFlags = setFlags{}
)

// setFlags holds the values bound to the `set` command's CLI flags.
type setFlags struct {
	CPUs     uint64
	Memory   uint64
	DiskSize uint64
}

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: setCmd,
	})

	flags := setCmd.Flags()

	cpusFlagName := "cpus"
	flags.Uint64Var(&setOptsFromFlags.CPUs, cpusFlagName, 0, "Number of CPUs")
	_ = setCmd.RegisterFlagCompletionFunc(cpusFlagName, completion.AutocompleteNone)

	memoryFlagName := "memory"
	flags.Uint64VarP(&setOptsFromFlags.Memory, memoryFlagName, "m", 0, "Memory in MiB")
	_ = setCmd.RegisterFlagCompletionFunc(memoryFlagName, completion.AutocompleteNone)

	diskSizeFlagName := "disk-size"
	flags.Uint64Var(&setOptsFromFlags.DiskSize, diskSizeFlagName, 0, "Disk size in GiB (can only be increased)")
	_ = setCmd.RegisterFlagCompletionFunc(diskSizeFlagName, completion.AutocompleteNone)
}

// setMachine applies the requested CPU and/or memory updates to an existing VM.
// It validates the requested values against host resources and requires at least
// one of --cpus or --memory to be provided.
func setMachine(cmd *cobra.Command, args []string) error {
	machineName := defaultMachineName
	if len(args) > 0 && len(args[0]) > 0 {
		machineName = args[0]
	}

	setOpts := define.SetOptions{}

	if cmd.Flags().Changed("cpus") {
		if setOptsFromFlags.CPUs == 0 {
			return fmt.Errorf("number of CPUs must be greater than 0")
		}
		if err := checkMaxCPUs(setOptsFromFlags.CPUs); err != nil {
			return err
		}
		setOpts.CPUs = &setOptsFromFlags.CPUs
	}
	if cmd.Flags().Changed("memory") {
		if setOptsFromFlags.Memory == 0 {
			return fmt.Errorf("memory must be greater than 0")
		}
		newMemory := strongunits.MiB(setOptsFromFlags.Memory)
		if err := checkMaxMemory(newMemory); err != nil {
			return err
		}
		setOpts.Memory = &newMemory
	}
	if cmd.Flags().Changed("disk-size") {
		if setOptsFromFlags.DiskSize == 0 {
			return fmt.Errorf("disk size must be greater than 0")
		}
		newDiskSize := strongunits.GiB(setOptsFromFlags.DiskSize)
		setOpts.DiskSize = &newDiskSize
	}

	if setOpts.CPUs == nil && setOpts.Memory == nil && setOpts.DiskSize == nil {
		return fmt.Errorf("at least one of --cpus, --memory or --disk-size must be specified")
	}

	vmProvider, err := provider2.GetProviderOrDefault(provider)
	if err != nil {
		return err
	}
	mc, _, err := shim.VMExists(machineName, []vmconfigs.VMProvider{vmProvider})
	if err != nil {
		return err
	}
	if mc == nil {
		return fmt.Errorf("VM %q does not exist", machineName)
	}
	if setOpts.DiskSize != nil {
		if err := checkNewDiskSize(*setOpts.DiskSize, mc.Resources.DiskSize); err != nil {
			return err
		}
	}

	return shim.Set(mc, vmProvider, setOpts)
}

// checkMaxCPUs compares requested CPUs to the host (runtime.NumCPU).
func checkMaxCPUs(requestedCPUs uint64) error {
	hostCPUs := uint64(runtime.NumCPU())
	if requestedCPUs > hostCPUs {
		return fmt.Errorf("requested number of CPUs (%d) greater than number of host CPUs (%d)", requestedCPUs, hostCPUs)
	}
	return nil
}

// checkMaxMemory gets the total system memory and compares it to the variable.
// If the variable is larger than the total memory, it returns an error.
func checkMaxMemory(newMem strongunits.MiB) error {
	memStat, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	if total := strongunits.B(memStat.Total); total < newMem.ToBytes() {
		return fmt.Errorf("requested amount of memory (%d MiB) greater than total system memory (%d MiB)", newMem, strongunits.ToMib(total))
	}
	return nil
}

func checkNewDiskSize(newDiskSize, currentDiskSize strongunits.GiB) error {
	if newDiskSize <= currentDiskSize {
		return fmt.Errorf("disk size must be greater than the current disk size (%d GiB)", currentDiskSize)
	}
	return nil
}
