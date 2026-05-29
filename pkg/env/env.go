package env

import (
	"fmt"
	"os"
	"path/filepath"

	"go.podman.io/podman/v6/pkg/machine/define"
	"go.podman.io/podman/v6/pkg/machine/vmconfigs"
	"go.podman.io/storage/pkg/homedir"
)

const (
	// Environment variables that third-party applications can set before
	// invoking the macadam binary to customize its behavior.
	EnvDataDir        = "MACADAM_DATA_DIR"
	EnvRuntimeDir     = "MACADAM_RUNTIME_DIR"
	EnvConnectionsDir = "MACADAM_CONNECTIONS_DIR"
	EnvToolName       = "MACADAM_TOOL_NAME"
)

// EnvironmentOptions controls the environment variables that macadam sets
// before invoking vendored podman machine code. Third-party applications
// can override these values by setting the MACADAM_* environment variables
// before launching the macadam binary.
type EnvironmentOptions struct {
	// DataDir controls PODMAN_DATA_DIR.
	// Relative paths are joined as <$XDG_DATA_HOME|$HOME/.local/share/containers>/<DataDir> (default behavior).
	// Absolute paths are used directly, bypassing the XDG base and "containers/" prefix.
	// Default: "macadam/machine"
	// Env override: MACADAM_DATA_DIR
	DataDir string

	// RuntimeDir controls PODMAN_RUNTIME_DIR. The provider type (e.g. "hyperv")
	// is appended automatically.
	// Relative paths are joined with the platform runtime base (e.g. /tmp/<uid>).
	// Absolute paths are used directly.
	// Default: "macadam"
	// Env override: MACADAM_RUNTIME_DIR
	RuntimeDir string

	// ConnectionsDir controls where the connections file is stored.
	// Relative paths are joined under <configHome>/ (e.g. ~/.config/<ConnectionsDir>/).
	// Absolute paths are used directly, bypassing the config home prefix.
	// Default: "macadam"
	// Env override: MACADAM_CONNECTIONS_DIR
	ConnectionsDir string

	// ToolName is the prefix used for WSL distros, Hyper-V registry entries,
	// machine name prefixing, and the connections file name
	// (<ToolName>-connections.json).
	// Default: "macadam"
	// Env override: MACADAM_TOOL_NAME
	ToolName string
}

func DefaultEnvironmentOptions() EnvironmentOptions {
	return EnvironmentOptions{
		DataDir:        envOrDefault(EnvDataDir, filepath.Join("macadam", "machine")),
		RuntimeDir:     envOrDefault(EnvRuntimeDir, "macadam"),
		ConnectionsDir: envOrDefault(EnvConnectionsDir, "macadam"),
		ToolName:       envOrDefault(EnvToolName, "macadam"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func connectionsFileName(toolName string) string {
	return toolName + "-connections.json"
}

func SetupEnvironment(provider vmconfigs.VMProvider, opts EnvironmentOptions) error {
	var connsDir string
	if filepath.IsAbs(opts.ConnectionsDir) {
		connsDir = opts.ConnectionsDir
	} else {
		configHome, err := homedir.GetConfigHome()
		if err != nil {
			return err
		}
		connsDir = filepath.Join(configHome, opts.ConnectionsDir)
	}

	connsFile := filepath.Join(connsDir, connectionsFileName(opts.ToolName))
	if err := os.Setenv("PODMAN_CONNECTIONS_CONF", connsFile); err != nil {
		return fmt.Errorf("setting PODMAN_CONNECTIONS_CONF: %w", err)
	}

	if err := os.Setenv("PODMAN_DATA_DIR", opts.DataDir); err != nil {
		return fmt.Errorf("setting PODMAN_DATA_DIR: %w", err)
	}

	runtimeDir := filepath.Join(opts.RuntimeDir, provider.VMType().String())
	if err := os.Setenv("PODMAN_RUNTIME_DIR", runtimeDir); err != nil {
		return fmt.Errorf("setting PODMAN_RUNTIME_DIR: %w", err)
	}

	if provider.VMType() == define.WSLVirt || provider.VMType() == define.HyperVVirt {
		if err := os.Setenv("PODMAN_TOOL_PREFIX", opts.ToolName); err != nil {
			return fmt.Errorf("setting PODMAN_TOOL_PREFIX: %w", err)
		}
	}

	return nil
}
