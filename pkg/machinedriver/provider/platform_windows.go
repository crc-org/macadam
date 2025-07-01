package provider

import (
	"fmt"

	"github.com/containers/podman/v5/pkg/machine/define"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
	wslPkg "github.com/containers/podman/v5/pkg/machine/wsl"
)

const wsl = "wsl"

func GetProviderOrDefault(name string) (vmconfigs.VMProvider, error) {
	if name == "" {
		name = define.WSLVirt.String()
	}
	switch name {
	case wsl:
		return new(wslPkg.WSLStubber), nil
	default:
		return nil, fmt.Errorf("unknown provider `%s`. Valid providers are: %v", name, GetProviders())
	}
}

func GetProviders() []string {
	return []string{wsl}
}

func GetDefaultProvider() string {
	return wsl
}
