//go:build windows

package hyperv

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/containers/podman/v5/pkg/machine"
	"github.com/containers/podman/v5/pkg/machine/hyperv/vsock"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
	"github.com/sirupsen/logrus"
)

func removeShares(mc *vmconfigs.MachineConfig) error {
	var removalErr error

	for _, mount := range mc.Mounts {
		if mount.VSockNumber == nil {
			// nothing to do if the vsock number was never defined
			continue
		}

		vsockReg, err := vsock.LoadHVSockRegistryEntry(*mount.VSockNumber)
		if err != nil {
			logrus.Debugf("Vsock %d for mountpoint %s does not have a valid registry entry, skipping removal", *mount.VSockNumber, mount.Target)
			continue
		}

		if err := vsockReg.Remove(); err != nil {
			if removalErr != nil {
				logrus.Errorf("Error removing vsock: %v", removalErr)
			}
			removalErr = fmt.Errorf("removing vsock %d for mountpoint %s: %w", *mount.VSockNumber, mount.Target, err)
		}
	}

	return removalErr
}

func startShares(mc *vmconfigs.MachineConfig) error {
	for _, mount := range mc.Mounts {
		var args []string
		cleanTarget := path.Clean(mount.Target)
		requiresChattr := !strings.HasPrefix(cleanTarget, "/home") && !strings.HasPrefix(cleanTarget, "/mnt")
		if requiresChattr {
			args = append(args, "sudo", "chattr", "-i", "/", "; ")
		}
		args = append(args, "sudo", "mkdir", "-p", strconv.Quote(cleanTarget), "; ")
		if requiresChattr {
			args = append(args, "sudo", "chattr", "+i", "/", "; ")
		}

		// just being protective here; in a perfect world, this cannot happen
		if mount.VSockNumber == nil {
			return errors.New("cannot start 9p shares with undefined vsock number")
		}

		if mc.CloudInit {
			// Mount using Unix socket created by systemd
			// The systemd service proxies vsock to /run/9p-<port>.sock
			unixSocketPath := fmt.Sprintf("/run/9p-%d.sock", *mount.VSockNumber)
			quotedTarget := strconv.Quote(cleanTarget)
			// Wait for socket with timeout (120 seconds max, checking every 0.4s = 300 iterations)
			mountOpts := "trans=unix,version=9p2000.L"
			if mount.ReadOnly {
				mountOpts += ",ro"
			}
			mountScript := fmt.Sprintf(`i=0; while [ ! -S %s ] && [ $i -lt 300 ]; do sleep 0.4; i=$((i+1)); done; [ -S %s ] || { echo "Timeout"; exit 1; }; sleep 2; mount -t 9p -o %s %s %s`, unixSocketPath, unixSocketPath, mountOpts, unixSocketPath, quotedTarget)
			// Quote the entire script so it survives strings.Join in the SSH function
			args = append(args, "sudo", "sh", "-c", fmt.Sprintf("'%s'", mountScript))
		} else {
			args = append(args, "sudo", "podman")
			if logrus.IsLevelEnabled(logrus.DebugLevel) {
				args = append(args, "--log-level=debug")
			}
			args = append(args, "machine", "client9p", fmt.Sprintf("%d", *mount.VSockNumber), strconv.Quote(mount.Target))
		}

		if err := machine.LocalhostSSHWithAddress(mc.SSH.RemoteUsername, mc.SSH.IdentityPath, mc.Name, mc.IPAddress, mc.SSH.Port, args); err != nil {
			return err
		}
	}
	return nil
}

func createShares(mc *vmconfigs.MachineConfig) (err error) {
	for _, mount := range mc.Mounts {
		testVsock, err := vsock.NewHVSockRegistryEntry(mc.Name, vsock.Fileserver)
		if err != nil {
			return err
		}
		mount.VSockNumber = &testVsock.Port
		logrus.Debugf("Going to share directory %s via 9p on vsock %d", mount.Source, testVsock.Port)
	}
	return nil
}
