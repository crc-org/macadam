package preflights

import (
	"bytes"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/containers/common/pkg/config"
	"github.com/containers/podman/v5/pkg/machine"
	"github.com/containers/podman/v5/pkg/machine/define"
	provider2 "github.com/containers/podman/v5/pkg/machine/provider"
)

// macadam/podman needs a gvproxy version which supports the --services
// argument
func CheckGvproxyVersion() error {
	provider, err := provider2.Get()
	if err != nil {
		return err
	}

	if provider.VMType() == define.WSLVirt {
		return nil
	}

	cfg, err := config.Default()
	if err != nil {
		return err
	}

	binary, err := cfg.FindHelperBinary(machine.ForwarderBinaryName, false)
	if err != nil {
		return err
	}

	cmd := exec.Command(binary, "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("failed to run gvproxy", "path", binary, "error", err)
	}
	if !bytes.Contains(out, []byte("-services")) {
		return fmt.Errorf("%s does not have support for the -services argument, please update to gvproxy v0.8.3 or newer", binary)
	}

	return nil
}
