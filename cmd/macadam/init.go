//go:build amd64 || arm64

package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	ldefine "github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/machine/define"
	"github.com/containers/podman/v5/pkg/machine/env"
	providerPodman "github.com/containers/podman/v5/pkg/machine/provider"
	"github.com/containers/podman/v5/pkg/machine/shim"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
	"github.com/crc-org/macadam/cmd/macadam/registry"
	"github.com/crc-org/macadam/pkg/imagepullers"
	macadam "github.com/crc-org/macadam/pkg/machinedriver"
	provider2 "github.com/crc-org/macadam/pkg/machinedriver/provider"
	"github.com/crc-org/macadam/pkg/preflights"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"
	"go.podman.io/common/pkg/completion"
	"go.podman.io/common/pkg/strongunits"
)

var (
	initCmd = &cobra.Command{
		Use:               "init [options] [IMAGE]",
		Short:             "Initialize a virtual machine",
		Long:              "Initialize a virtual machine",
		RunE:              initMachine,
		Args:              cobra.MaximumNArgs(1),
		Example:           `macadam init image.raw`,
		ValidArgsFunction: completion.AutocompleteNone,
	}

	initOptsFromFlags  = define.InitOptions{}
	initOptionalFlags  = InitOptionalFlags{}
	defaultMachineName = "macadam"
	// now                bool
)

// Flags which have a meaning when unspecified that differs from the flag default
type InitOptionalFlags struct {
	UserModeNetworking bool
}

// maxMachineNameSize is set to thirty to limit huge machine names primarily
// because macOS has a much smaller file size limit.
const maxMachineNameSize = 30

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: initCmd,
	})

	flags := initCmd.Flags()

	MachineNameFlagName := "name"
	flags.StringVar(&initOptsFromFlags.Name, MachineNameFlagName, defaultMachineName, "Name for the machine")
	_ = initCmd.RegisterFlagCompletionFunc(MachineNameFlagName, completion.AutocompleteDefault)

	SSHIdentityPathFlagName := "ssh-identity-path"
	flags.StringVar(&initOptsFromFlags.SSHIdentityPath, SSHIdentityPathFlagName, "", "Path to the SSH private key to use to access the machine")
	_ = initCmd.RegisterFlagCompletionFunc(SSHIdentityPathFlagName, completion.AutocompleteDefault)

	UsernameFlagName := "username"
	flags.StringVar(&initOptsFromFlags.Username, UsernameFlagName, "core", "Username used in image")
	_ = initCmd.RegisterFlagCompletionFunc(UsernameFlagName, completion.AutocompleteDefault)

	cpusFlagName := "cpus"
	flags.Uint64Var(&initOptsFromFlags.CPUS, cpusFlagName, 2, "Number of CPUs")
	_ = initCmd.RegisterFlagCompletionFunc(cpusFlagName, completion.AutocompleteNone)

	diskSizeFlagName := "disk-size"
	flags.Uint64Var(&initOptsFromFlags.DiskSize, diskSizeFlagName, 20, "Disk size in GiB")
	_ = initCmd.RegisterFlagCompletionFunc(diskSizeFlagName, completion.AutocompleteNone)

	memoryFlagName := "memory"
	flags.Uint64VarP(&initOptsFromFlags.Memory, memoryFlagName, "m", 4096, "Memory in MiB")
	_ = initCmd.RegisterFlagCompletionFunc(memoryFlagName, completion.AutocompleteNone)

	CloudInitPathFlagName := "cloud-init"
	flags.StringSliceVarP(&initOptsFromFlags.CloudInitPaths, CloudInitPathFlagName, "", []string{}, "Path to user-data, meta-data and network-config cloud-init configuration files")
	_ = initCmd.RegisterFlagCompletionFunc(CloudInitPathFlagName, completion.AutocompleteDefault)

	// User-mode networking flag is only available on Windows (HyperV-only)
	if runtime.GOOS == "windows" {
		userModeNetFlagName := "user-mode-networking"
		flags.BoolVar(&initOptionalFlags.UserModeNetworking, userModeNetFlagName, false,
			"Whether this machine should use user-mode networking, routing traffic through a host user-space process (Hyperv-only, requires --provider=hyperv)")
	}

	VolumeFlagName := "volume"
	flags.StringArrayVarP(&initOptsFromFlags.Volumes, VolumeFlagName, "v", []string{}, "Volumes to mount, source:target")
	_ = initCmd.RegisterFlagCompletionFunc(VolumeFlagName, completion.AutocompleteDefault)

	/* flags := initCmd.Flags()
	cfg := registry.PodmanConfig()

	flags.BoolVar(
		&now,
		"now", false,
		"Start machine now",
	)
	timezoneFlagName := "timezone"
	defaultTz := cfg.ContainersConfDefaultsRO.TZ()
	if len(defaultTz) < 1 {
		defaultTz = "local"
	}
	flags.StringVar(&initOpts.TimeZone, timezoneFlagName, defaultTz, "Set timezone")
	_ = initCmd.RegisterFlagCompletionFunc(timezoneFlagName, completion.AutocompleteDefault)

	flags.BoolVar(
		&initOpts.ReExec,
		"reexec", false,
		"process was rexeced",
	)
	_ = flags.MarkHidden("reexec")

	UsernameFlagName := "username"
	flags.StringVar(&initOpts.Username, UsernameFlagName, cfg.ContainersConfDefaultsRO.Machine.User, "Username used in image")
	_ = initCmd.RegisterFlagCompletionFunc(UsernameFlagName, completion.AutocompleteDefault)

	ImageFlagName := "image"
	flags.StringVar(&initOpts.Image, ImageFlagName, cfg.ContainersConfDefaultsRO.Machine.Image, "Bootable image for machine")
	_ = initCmd.RegisterFlagCompletionFunc(ImageFlagName, completion.AutocompleteDefault)

	// Deprecate image-path option, use --image instead
	ImagePathFlagName := "image-path"
	flags.StringVar(&initOpts.Image, ImagePathFlagName, cfg.ContainersConfDefaultsRO.Machine.Image, "Bootable image for machine")
	_ = initCmd.RegisterFlagCompletionFunc(ImagePathFlagName, completion.AutocompleteDefault)
	if err := flags.MarkDeprecated(ImagePathFlagName, "use --image instead"); err != nil {
		logrus.Error("unable to mark image-path flag deprecated")
	}

	USBFlagName := "usb"
	flags.StringArrayVarP(&initOpts.USBs, USBFlagName, "", []string{},
		"USB Host passthrough: bus=$1,devnum=$2 or vendor=$1,product=$2")
	_ = initCmd.RegisterFlagCompletionFunc(USBFlagName, completion.AutocompleteDefault)

	VolumeDriverFlagName := "volume-driver"
	flags.String(VolumeDriverFlagName, "", "Optional volume driver")
	_ = initCmd.RegisterFlagCompletionFunc(VolumeDriverFlagName, completion.AutocompleteDefault)
	if err := flags.MarkDeprecated(VolumeDriverFlagName, "will be ignored"); err != nil {
		logrus.Error("unable to mark volume-driver flag deprecated")
	}

	IgnitionPathFlagName := "ignition-path"
	flags.StringVar(&initOpts.IgnitionPath, IgnitionPathFlagName, "", "Path to ignition file")
	_ = initCmd.RegisterFlagCompletionFunc(IgnitionPathFlagName, completion.AutocompleteDefault)

	rootfulFlagName := "rootful"
	flags.BoolVar(&initOpts.Rootful, rootfulFlagName, false, "Whether this machine should prefer rootful container execution") */
}

// cleanupOrphanedFiles cleans up orphaned files for a given machine
func cleanupOrphanedFiles(vmProvider vmconfigs.VMProvider, name string) {
	dirs, err := env.GetMachineDirs(vmProvider.VMType())
	if err != nil {
		return
	}

	mc, err := vmconfigs.LoadMachineByName(name, dirs)
	if err != nil {
		return
	}

	machines, err := providerPodman.GetAllMachinesAndRootfulness()
	if err != nil {
		return
	}

	rmFiles, genericRm, err := mc.Remove(machines, false, false)
	if err != nil {
		slog.Debug(fmt.Sprintf("failed to remove machines files of a previous machine having the same name %q. Error: %v", name, err))
	}

	if len(rmFiles) > 0 {
		if err := genericRm(); err != nil {
			slog.Warn(fmt.Sprintf("found orphaned file of a previous machine having the same name %q. Tried to clean the environment but failed to remove old machines files. %v", name, err))
		}
	}
}

func initMachine(cmd *cobra.Command, args []string) error {
	vmProvider, err := provider2.GetProviderOrDefault(provider)
	if err != nil {
		return err
	}

	diskImage := ""
	if len(args) > 0 {
		diskImage = args[0]
	}

	machineName := initOptsFromFlags.Name
	if len(machineName) > maxMachineNameSize {
		return fmt.Errorf("machine name %q must be %d characters or less", machineName, maxMachineNameSize)
	}

	if !ldefine.NameRegex.MatchString(machineName) {
		return fmt.Errorf("invalid name %q: %w", machineName, ldefine.RegexError)
	}

	mc, _, err := shim.VMExists(machineName, []vmconfigs.VMProvider{vmProvider})
	if err != nil {
		return err
	}
	if mc != nil {
		// VM exists, return error
		return fmt.Errorf("VM %s already exists", machineName)
	}

	// VM doesn't exist, check if there are orphaned files to clean up
	// it may happen that a machine has been deleted externally (e.g. through the Hyper-v Manager)
	// and podman was not aware of it. If the user wants to create a new machine with the same name,
	// we should clean up the orphaned files.
	cleanupOrphanedFiles(vmProvider, machineName)

	initOpts := macadam.DefaultInitOpts(machineName)
	if cmd.Flags().Changed("user-mode-networking") {
		initOpts.UserModeNetworking = &initOptionalFlags.UserModeNetworking
	}

	if err := preflights.RunPreflights(vmProvider, initOpts.UserModeNetworking); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	// Check if the disk image exists and is not larger than the specified disk size
	if diskImage == "" {
		return fmt.Errorf("disk image is required")
	}

	fileInfo, err := os.Stat(diskImage)
	if err != nil {
		return fmt.Errorf("failed to stat disk image %q: %w", diskImage, err)
	}

	diskSizeInBytes := int64(strongunits.GiB(initOptsFromFlags.DiskSize).ToBytes())
	if fileInfo.Size() > diskSizeInBytes {
		return fmt.Errorf("disk image %s (size: %s) is larger than the expected maximum size of %s",
			diskImage, units.HumanSize(float64(fileInfo.Size())), units.HumanSize(float64(diskSizeInBytes)))
	}

	puller := imagepullers.NewNoopImagePuller(machineName, vmProvider.VMType())

	initOpts.ImagePuller = puller
	initOpts.ImagePuller.SetSourceURI(diskImage)
	initOpts.Name = machineName
	initOpts.Image = diskImage
	initOpts.CPUS = initOptsFromFlags.CPUS
	initOpts.DiskSize = initOptsFromFlags.DiskSize
	initOpts.Memory = initOptsFromFlags.Memory
	initOpts.SSHIdentityPath = initOptsFromFlags.SSHIdentityPath
	initOpts.Username = initOptsFromFlags.Username
	initOpts.Volumes = initOptsFromFlags.Volumes
	initOpts.CloudInit = true // this should be calculated based on the image we want to start ??
	initOpts.CloudInitPaths = initOptsFromFlags.CloudInitPaths
	initOpts.Capabilities = &define.MachineCapabilities{
		HasReadyUnit:   false,
		ForwardSockets: false,
	}

	/*
		_, _, err = shim.VMExists(machineName, []vmconfigs.VMProvider{provider})
		if err == nil {
			return fmt.Errorf("machine %q already exists", machineName)
		}
	*/
	return shim.Init(*initOpts, vmProvider)
}
