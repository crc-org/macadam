package provider

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"go.podman.io/podman/v6/pkg/machine/define"
	hypervPkg "go.podman.io/podman/v6/pkg/machine/hyperv"
	"go.podman.io/podman/v6/pkg/machine/vmconfigs"
	wslPkg "go.podman.io/podman/v6/pkg/machine/wsl"
)

var defaultProvider = define.WSLVirt

func GetProviderOrDefault(name string) (vmconfigs.VMProvider, error) {
	resolvedVMType, err := define.ParseVMType(name, defaultProvider)
	if err != nil {
		return nil, err
	}

	logrus.Debugf("Using macadam with `%s` virtualization provider", resolvedVMType.String())
	switch resolvedVMType {
	case define.WSLVirt:
		return new(wslPkg.WSLStubber), nil
	case define.HyperVVirt:
		return new(hypervPkg.HyperVStubber), nil
	default:
		return nil, fmt.Errorf("unknown provider `%s`. Valid providers are: %v", name, GetProviders())
	}
}

func GetProviders() []string {
	return []string{define.WSLVirt.String(), define.HyperVVirt.String()}
}

func GetDefaultProvider() string {
	return defaultProvider.String()
}
