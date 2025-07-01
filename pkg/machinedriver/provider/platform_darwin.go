package provider

import (
	"fmt"

	"github.com/containers/podman/v5/pkg/machine/applehv"
	"github.com/containers/podman/v5/pkg/machine/define"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
)

const appleHV = "applehv"

func GetProviderOrDefault(name string) (vmconfigs.VMProvider, error) {
	if name == "" {
		name = define.AppleHvVirt.String()
	}
	switch name {
	case appleHV:
		return new(applehv.AppleHVStubber), nil
	default:
		return nil, fmt.Errorf("unknown provider `%s`. Valid providers are: %v", name, GetProviders())
	}
}

func GetProviders() []string {
	return []string{appleHV}
}

func GetDefaultProvider() string {
	return appleHV
}
