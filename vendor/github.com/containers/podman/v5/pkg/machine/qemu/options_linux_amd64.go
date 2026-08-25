//go:build linux && amd64

package qemu

import (
	"path/filepath"

	"go.podman.io/storage/pkg/fileutils"
)

var qemuCommand = []string{"qemu-system-x86_64", "qemu-kvm"}

func (q *QEMUStubber) addArchOptions(_ *setNewMachineCMDOpts) []string {
	opts := []string{
		"-accel", "kvm",
		"-cpu", "host",
		"-M", "memory-backend=mem",
	}
	if q.Firmware == "uefi" {
		opts = append(opts, "-bios", getQemuUefiFile("OVMF.fd"))
	}
	return opts
}

func getQemuUefiFile(name string) string {
	dirs := []string{
		"/usr/share/OVMF",
		"/usr/share/ovmf",
		"/usr/share/edk2/ovmf",
	}
	for _, dir := range dirs {
		full := filepath.Join(dir, name)
		if err := fileutils.Exists(full); err == nil {
			return full
		}
	}
	return name
}
