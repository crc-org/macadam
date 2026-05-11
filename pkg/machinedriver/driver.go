/*
Copyright 2021, Red Hat, Inc - All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package macadam

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/containers/podman/v5/pkg/machine"
	"github.com/containers/podman/v5/pkg/machine/define"
	"github.com/containers/podman/v5/pkg/machine/env"
	"github.com/containers/podman/v5/pkg/machine/shim"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
	"github.com/crc-org/machine/libmachine/drivers"
	"github.com/crc-org/machine/libmachine/state"
)

const (
	DefaultMemory  = 8192
	DefaultCPUs    = 4
	DefaultSSHUser = "core"
)

const (
	// from "github.com/crc-org/crc/v2/pkg/crc/constants"
	DaemonVsockPort = 1024
)

type Driver struct {
	*drivers.VMDriver
	VirtioNet bool

	// TODO: add configuration for this in podman machine
	VsockPath       string
	DaemonVsockPort uint
	QemuGAVsockPort uint

	vmConfig   *vmconfigs.MachineConfig
	vmProvider vmconfigs.VMProvider
}

// this func should return the driver by using the provider and machineName
func GetDriverByProviderAndMachineName(provider vmconfigs.VMProvider, machineName string) (*Driver, error) {
	mc, _, err := shim.VMExists(machineName, []vmconfigs.VMProvider{provider})
	if err != nil {
		return nil, err
	}
	if mc == nil {
		return nil, fmt.Errorf("VM %q does not exist", machineName)
	}

	return &Driver{
		VMDriver: &drivers.VMDriver{
			BaseDriver: &drivers.BaseDriver{
				MachineName: machineName,
			},
			CPU:    DefaultCPUs,
			Memory: DefaultMemory,
		},
		// needed when loading a VM which was created before
		// DaemonVsockPort was introduced
		DaemonVsockPort: DaemonVsockPort,
		vmConfig:        mc,

		vmProvider: provider,
	}, nil
}

func DefaultInitOpts(machineName string) *define.InitOptions {
	initOpts := define.InitOptions{}
	// defaults from cmd/podman/machine/init.go
	initOpts.Name = machineName

	initOpts.CPUS = uint64(DefaultCPUs)
	//initOpts.DiskSize = uint64(???)
	initOpts.Memory = uint64(DefaultMemory)
	initOpts.TimeZone = ""
	initOpts.ReExec = false
	/*
		initOpts.Username = defaultConfig.Machine.User
		initOpts.Image = defaultConfig.Machine.Image
		initOpts.Volumes = defaultConfig.Machine.Volumes.Get()
	*/
	initOpts.Username = DefaultSSHUser
	//initOpts.SSHIdentityPath = d.VMDriver.SSHConfig.IdentityPath
	/* if d.VMDriver.SSHConfig.RemoteUsername != "" {
		initOpts.Username = d.VMDriver.SSHConfig.RemoteUsername
	} */
	initOpts.Image = "" // cli mode will most likely not copy images to a central place
	initOpts.Volumes = []string{}
	initOpts.USBs = []string{}
	initOpts.IgnitionPath = ""
	initOpts.Rootful = false

	return &initOpts
}

func Start(vmConfig *vmconfigs.MachineConfig, vmProvider vmconfigs.VMProvider) error {
	machineName := vmConfig.Name
	dirs, err := env.GetMachineDirs(vmProvider.VMType())
	if err != nil {
		return err
	}

	fmt.Printf("Starting machine %q\n", machineName)

	startOpts := machine.StartOptions{
		// Running the e2e tests on ubuntu requires slightly more time for ssh to come up
		MaxBackoffs: 10,
		// The ForwardSockets capabilities roughly indicates if we have a podman machine or a generic VM.
		// For generic VMs, we don’t want to print these `info` messages as they are podman-centric.
		NoInfo: !vmConfig.Capabilities.GetForwardSockets(),
		Quiet:  false,
	}
	slog.Debug("SSH config", "port", vmConfig.SSH.Port, "username", vmConfig.SSH.RemoteUsername, "identity-path", vmConfig.SSH.IdentityPath)

	if err := shim.Start(vmConfig, vmProvider, dirs, startOpts); err != nil {
		return err
	}
	fmt.Printf("Machine %q started successfully\n", machineName)
	//newMachineEvent(events.Start, events.Event{Name: vmName})
	return nil
}

func podmanStatusToCrcState(status define.Status) state.State {
	switch status {
	case define.Running:
		return state.Running
	case define.Stopped:
		return state.Stopped
	case define.Starting:
		return state.Running
	case define.Unknown:
		return state.Error
	}

	// unknown state
	return state.Error
}

// GetState returns the state that the host is in (running, stopped, etc)
func (d *Driver) GetState() (state.State, error) {
	if d.vmConfig == nil {
		return state.Stopped, nil
	}
	status, err := d.vmProvider.State(d.vmConfig, false)
	if err != nil {
		return state.Error, err
	}
	return podmanStatusToCrcState(status), nil
	// piggy back on podman machine state
	//return state.Error, fmt.Errorf("GetState() unimplemented")
}

func (d *Driver) RemoveWithOptions(opts machine.RemoveOptions) error {
	machineName := d.vmConfig.Name
	fmt.Printf("Removing machine %q\n", machineName)
	dirs, err := env.GetMachineDirs(d.vmProvider.VMType())
	if err != nil {
		return err
	}

	if err := shim.Remove(d.vmConfig, d.vmProvider, dirs, opts); err != nil {
		/* don’t print anything if the user cancelled the removal */
		if errors.Is(err, shim.ErrRemoveUserCancelled) {
			return nil
		}
		return err
	}
	//newMachineEvent(events.Remove, events.Event{Name: vmName})
	fmt.Printf("Machine %q removed successfully\n", machineName)
	return nil
}

// Stop a host gracefully
func (d *Driver) Stop() error {
	fmt.Printf("Stopping machine %q\n", d.vmConfig.Name)
	if err := d.stop(false); err != nil {
		return err
	}
	//newMachineEvent(events.Stop, events.Event{Name: vmName})
	fmt.Printf("Machine %q stopped successfully\n", d.vmConfig.Name)
	return nil
}

func (d *Driver) stop(hardStop bool) error {
	dirs, err := env.GetMachineDirs(d.vmProvider.VMType())
	if err != nil {
		return err
	}

	if err := shim.Stop(d.vmConfig, d.vmProvider, dirs, hardStop); err != nil {
		return err
	}
	//newMachineEvent(events.Remove, events.Event{Name: vmName})
	return nil
}

func (d *Driver) GetVmConfig() *vmconfigs.MachineConfig {
	return d.vmConfig
}

func (d *Driver) GetVMType() define.VMType {
	return d.vmProvider.VMType()
}

func List(vmstubbers []vmconfigs.VMProvider) ([]*Driver, error) {
	var (
		drivers []*Driver
	)

	for _, s := range vmstubbers {
		dirs, err := env.GetMachineDirs(s.VMType())
		if err != nil {
			return nil, err
		}
		mcs, err := vmconfigs.LoadMachinesInDir(dirs)
		if err != nil {
			return nil, err
		}
		for name := range mcs {
			driver, err := GetDriverByProviderAndMachineName(s, name)
			if err != nil {
				return nil, err
			}

			drivers = append(drivers, driver)
		}
	}

	return drivers, nil
}
