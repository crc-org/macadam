//go:build !windows && !darwin

package provider

import (
	"fmt"

	"github.com/containers/podman/v5/pkg/machine/define"
	qemuPkg "github.com/containers/podman/v5/pkg/machine/qemu"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
)

const qemu = "qemu"

func GetProviderOrDefault(name string) (vmconfigs.VMProvider, error) {
	if name == "" {
		name = define.QemuVirt.String()
	}
	switch name {
	case qemu:
		return new(qemuPkg.QEMUStubber), nil
	default:
		return nil, fmt.Errorf("unknown provider `%s`. Valid providers are: %v", name, GetProviders())
	}
}

func GetProviders() []string {
	return []string{qemu}
}

func GetDefaultProvider() string {
	return qemu
}
