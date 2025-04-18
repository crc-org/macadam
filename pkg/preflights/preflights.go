package preflights

import (
	"bytes"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"

	"github.com/containers/common/pkg/config"
	"github.com/containers/podman/v5/pkg/machine"
	"github.com/containers/podman/v5/pkg/machine/define"
	provider2 "github.com/containers/podman/v5/pkg/machine/provider"
)

func RunPreflights() error {
	if err := checkGvproxyVersion(); err != nil {
		return fmt.Errorf("invalid gvproxy binary: %w", err)
	}

	if err := checkVfkitVersion(); err != nil {
		return fmt.Errorf("invalid vfkit binary: %w", err)
	}

	if err := checkSupportedProviders(); err != nil {
		return err
	}

	return nil
}

// macadam/podman needs a gvproxy version which supports the --services
// argument
func checkGvproxyVersion() error {
	provider, err := provider2.Get()
	if err != nil {
		return err
	}

	if provider.VMType() == define.WSLVirt {
		return nil
	}
	if err := checkBinaryArg(machine.ForwarderBinaryName, "-services"); err != nil {
		return fmt.Errorf("%w, please update to gvproxy v0.8.3 or newer", err)
	}
	return nil
}

// macadam/podman needs a vfkit binary which supports the --cloud-init
// argument to inject ssh keys in RHEL cloud images
func checkVfkitVersion() error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	if err := checkBinaryArg("vfkit", "--cloud-init"); err != nil {
		return fmt.Errorf("%w, please update to vfkit from git main", err)
	}
	return nil
}

func checkSupportedProviders() error {
	provider, err := provider2.Get()
	if err != nil {
		return err
	}
	vmType := provider.VMType()
	switch vmType {
	case define.HyperVVirt, define.LibKrun:
		return fmt.Errorf("%s VM provider is unsupported, only wsl2 on Windows, vfkit on macOS and qemu on linux are supported", vmType.String())
	default:
		return nil
	}
}

func checkBinaryArg(binaryName, arg string) error {
	cfg, err := config.Default()
	if err != nil {
		return err
	}

	binary, err := cfg.FindHelperBinary(binaryName, false)
	if err != nil {
		return err
	}

	cmd := exec.Command(binary, "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("failed to run binary", "path", binary, "error", err)
	}
	if !bytes.Contains(out, []byte(arg)) {
		return fmt.Errorf("%s does not have support for the %s argument", binary, arg)
	}

	return nil
}
