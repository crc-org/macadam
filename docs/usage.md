# Macadam User Guide

Macadam is a cross-platform virtualization tool which is used for creating and managing virtual machines across different operating systems and hypervisors. 

> **Note:** Multiple machines can be created and run simultaneously.

## Overview

Macadam provides a unified interface for virtual machine management with platform-specific hypervisor support:

## Command Reference

### System Validation

#### `macadam preflight`

Performs comprehensive system validation to ensure all required components are properly configured.

**Validation checks include:**
- `gvproxy` version compatibility verification:
  - Performed on all operating systems except WSL
- `vfkit` version requirements confirmation:
  - Only performed on macOS systems utilizing appleHV
- `krunkit` version requirements confirmation:
  - Only performed on macOS systems utilizing libkrun

**Usage:**
```bash
macadam preflight
```

### Virtual Machine Management

#### `macadam init`
The `init` command initializes a new virtual machine instance using a 
specified disk image. This command requires cloud-init compatible images 
for proper virtual machine configuration such as Fedora Cloud.

**Image Requirements:**
- Supported formats by OS/Hypervisor:
  - WSL2: tar.gz, wsl
  - Hyper-V: vhd, vhdx
  - Linux, macOS: raw, qcow2
- Must be cloud-init compatible

The `macadam init` command accepts a single argument specifying the path to the source image. This should be a cloud-init compatible image. The provided image is copied to the containers config directory. The copied image is then used to boot the virtual machine. The command leverages podman machine initialization code underneath to initialize the VM.

**Usage:**
```bash
macadam init <path-to-image>
```

**Example:**
```bash
macadam init fedora-cloud.raw
```

**Flags:**

- `--name`: Specifies the name for the virtual machine. Must be 30 characters or less. Defaults to `macadam` if not provided.

- `--cpus`: Sets the number of CPU cores allocated to the virtual machine. Defaults to 2 if not specified.

- `--memory`: Configures the amount of memory (in MiB) allocated to the virtual machine. Defaults to 4096 MiB if not specified.

- `--disk-size`: Sets the disk size (in GiB) for the virtual machine. Defaults to 20 GiB if not specified.

- `--ssh-identity-path`: Path to the SSH private key to use to access the machine. If not provided, a macadam-specific SSH key is generated if needed, and shared by all VMs.

- `--username`: Sets the username for the virtual machine. Defaults to "core" if not specified.

#### `macadam start`

The `start` command starts an existing virtual machine that has been previously initialized. It accepts an optional machine name argument. If no name is provided, it defaults to starting the machine named `macadam`.

When `start` is called, it initializes a set of default options based on the configuration provided during `macadam init`. The command uses user-mode networking by default, although on Hyper-V, system-mode networking is used by default. The appropriate VM provider is automatically determined based on the operating system: WSL2 on Windows, AppleHV on macOS, and QEMU on Linux. However, users can specify the VM provider using the --provider flag. The valid provider values can be seen by reading the output of `macadam --help`. Note that the ability to choose a provider is limited: on Windows, users can select between WSL2 and Hyper-V, and on macOS, they can choose between AppleHV and LibKrun.

Platform-specific provider support:
- **macOS**: `applehv`, `libkrun`
- **Linux**: `qemu`
- **Windows**: `wsl2`, `hyperv`

The virtual machine starts with the configuration options specified during initialization.

**Usage:**

```bash
macadam start
```

#### `macadam stop`

The `stop` command stops a running virtual machine. It accepts an optional machine name argument. If no name is provided, it defaults to stopping the machine named `macadam`.

**Usage:**

```bash
macadam stop
```
The provider is automatically determined by your operating system if you don't specify it using the `--provider` flag. When `stop` is called, the machine is gracefully shut down using the appropriate method for your platform.

#### `macadam inspect`

The `macadam inspect` command provides detailed information about one or more virtual machines. You can specify a list of machine names as arguments; if no names are given, it defaults to inspecting the machine named `macadam`.

When you run `macadam inspect`, the tool first determines the appropriate provider for your platform (for example, `applehv` on macOS), or uses the one you specified with the `--provider` flag. It then retrieves the configuration for each specified machine that was started with that provider. For every machine, `macadam inspect` outputs comprehensive details, including resource allocation, SSH configuration, and the current state of the machine.

```bash
macadam inspect vm1 vm2...
```

The output of inspect shows the information in json format.

#### `macadam list`

The `macadam list` command displays all virtual machines that have been created. When executed, it automatically detects the current provider, or uses the one specified by the `--provider` flag, and then loads the machines. The resulting list is sorted by the most recent activity (last run time).

You can customize the output format using the `--format` flag. For example, specifying `--format json` will present the list in JSON format.

**Usage:**

```bash
macadam list
```
#### `macadam ssh`

The `macadam ssh` command allows you to connect to a running virtual machine via SSH. During `macadam init`, either the default or a user-provided SSH key is injected into the VM using cloud-init. This key is then used for authentication when connecting to the VM.

You can use `macadam ssh` with an optional `machine-name` argument, followed by a command you wish to execute in the VM. If no machine name is specified, `macadam` will attempt to SSH into the machine named `macadam` by default.

**Usage:**

```bash
macadam ssh
```

You can use the `--username` flag with `macadam ssh` to specify a different SSH username when connecting to the virtual machine. The default username is `core` which can be overridden using this flag. For example:

```bash
macadam ssh --username test
```

#### `macadam rm`

The `macadam rm` command removes an existing virtual machine. It accepts an optional machine name argument. If no name is provided, it defaults to removing the machine named `macadam`.

When you run `macadam rm`, the command will remove the virtual machine configuration and associated files. By default, the command will prompt for confirmation before removing the machine.

**Usage:**

```bash
macadam rm [MACHINE]
```

**Example:**

```bash
macadam rm vm1
```

**Flags:**

- `--force` (`-f`): Stop and do not prompt before removing the machine. This flag forces the removal without user confirmation.

**Example with force flag:**

```bash
macadam rm --force vm1
```

## Storage Organization

Macadam stores images, configuration, and runtime data in separate locations on your system.

- **VM Images:**  
   VM disk images are stored in `~/.local/share/containers/macadam/machine/`. Each hypervisor has its own directory within this location, and the disk images for each provider can be found in these directories.

- **VM Configs:**  
  Configuration files are located in `~/.config/containers/macadam/machine/`. These files contain settings such as CPU, memory, disk size, and SSH configuration.

- **Runtime Data:**  
  Runtime state and temporary files are stored in `$TMPDIR/macadam/`. This directory contains relevant runtime data, such as socket files.
