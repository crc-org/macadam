package preflights

import (
	"bytes"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/containers/podman/v5/pkg/machine/define"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
	"go.podman.io/common/pkg/config"
)

func RunPreflights(provider vmconfigs.VMProvider, userModeNetworking *bool) error {
	if err := validateOptions(provider, userModeNetworking); err != nil {
		return err
	}

	if err := checkGvproxyVersion(provider, userModeNetworking); err != nil {
		return fmt.Errorf("invalid gvproxy binary: %w", err)
	}

	if err := checkVfkitVersion(provider); err != nil {
		return fmt.Errorf("invalid vfkit binary: %w", err)
	}

	if err := checkKrunKitAvailability(provider); err != nil {
		return fmt.Errorf("missing krunkit binary: %w", err)
	}

	return nil
}

func validateOptions(provider vmconfigs.VMProvider, userModeNetworking *bool) error {
	if provider.VMType() == define.WSLVirt && userModeNetworking != nil && *userModeNetworking {
		return fmt.Errorf("user-mode networking is not supported on WSL. Please run the command without the --user-mode-networking flag")
	}
	return nil
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
