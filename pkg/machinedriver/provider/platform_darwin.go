package provider

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/sirupsen/logrus"
	"go.podman.io/podman/v6/pkg/machine/applehv"
	"go.podman.io/podman/v6/pkg/machine/define"
	"go.podman.io/podman/v6/pkg/machine/libkrun"
	"go.podman.io/podman/v6/pkg/machine/vmconfigs"
)

var defaultProvider = define.AppleHvVirt

func GetProviderOrDefault(name string) (vmconfigs.VMProvider, error) {
	resolvedVMType, err := define.ParseVMType(name, defaultProvider)
	if err != nil {
		return nil, err
	}

	logrus.Debugf("Using macadam with `%s` virtualization provider", resolvedVMType.String())
	switch resolvedVMType {
	case define.AppleHvVirt:
		return new(applehv.AppleHVStubber), nil
	case define.LibKrun:
		if runtime.GOARCH == "amd64" {
			return nil, errors.New("libkrun is not supported on Intel based machines. Please revert to the applehv provider")
		}
		return new(libkrun.LibKrunStubber), nil
	default:
		return nil, fmt.Errorf("unknown provider `%s`. Valid providers are: %v", name, GetProviders())
	}
}

func GetProviders() []string {
	configs := []string{define.AppleHvVirt.String()}
	if runtime.GOARCH == "arm64" {
		configs = append(configs, define.LibKrun.String())
	}
	return configs
}

func GetDefaultProvider() string {
	return defaultProvider.String()
}
