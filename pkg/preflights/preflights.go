package preflights

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containers/podman/v5/pkg/machine/define"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
	"github.com/sirupsen/logrus"
)

const (
	// Token prefix for looking for helper binary under $BINDIR
	bindirPrefix = "$BINDIR"
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
	binary, err := findBinary(binaryName)
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

// findBindir returns the directory of the current executable.
// Based on github.com/containers/common/pkg/config/config.go#findBindir
func findBindir() string {
	execPath, err := os.Executable()
	if err == nil {
		// Resolve symbolic links to find the actual binary file path.
		execPath, err = filepath.EvalSymlinks(execPath)
	}
	if err != nil {
		// If failed to find executable (unlikely to happen), warn about it.
		logrus.Warnf("Failed to find $BINDIR: %v", err)
		return ""
	}
	return filepath.Dir(execPath)
}

// findBinary searches for a binary in the configured directories.
// Based on github.com/containers/common/pkg/config/config.go#FindHelperBinary
func findBinary(name string) (string, error) {
	dirList := getBinariesDirs()
	// If set, search this directory first. This is used in testing.
	if dir, found := os.LookupEnv("CONTAINERS_HELPER_BINARY_DIR"); found {
		dirList = append([]string{dir}, dirList...)
	}

	for _, path := range dirList {
		if path == bindirPrefix || strings.HasPrefix(path, bindirPrefix+string(filepath.Separator)) {
			// Calculate the path to the executable with a $BINDIR prefix.
			bindirPath := findBindir()
			// If there's an error, don't stop the search for the helper binary.
			// findBindir() will have warned once during the first failure.
			if bindirPath == "" {
				continue
			}
			// Replace the $BINDIR prefix with the path to the directory of the current binary.
			if path == bindirPrefix {
				path = bindirPath
			} else {
				path = filepath.Join(bindirPath, strings.TrimPrefix(path, bindirPrefix+string(filepath.Separator)))
			}
		}
		// Absolute path will force exec.LookPath to check for binary existence instead of lookup everywhere in PATH
		if abspath, err := filepath.Abs(filepath.Join(path, name)); err == nil {
			// exec.LookPath from absolute path on Unix is equal to os.Stat + IsNotDir + check for executable bits in FileMode
			// exec.LookPath from absolute path on Windows is equal to os.Stat + IsNotDir for `file.ext` or loops through extensions from PATHEXT for `file`
			if lp, err := exec.LookPath(abspath); err == nil {
				return lp, nil
			}
		}
	}

	return "", fmt.Errorf("could not find %q in one of %v", name, dirList)
}
