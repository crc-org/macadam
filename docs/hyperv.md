# Running Macadam with Hyper-V on Windows

Macadam supports Hyper-V as a virtual machine provider on Windows. Unlike WSL2,
the Hyper-V provider requires a one-time host preparation so that normal
(non-administrator) users can create, start, stop, and remove virtual machines.

This page explains what the preparation does, why it is needed, and how the
different workflows behave depending on whether the host has been prepared in
advance.

## Background

Hyper-V virtual machines communicate with the host through *vsock* (virtual
socket) connections. Each connection requires a registry entry under:

```
HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Virtualization\GuestCommunicationServices
```

Writing to `HKEY_LOCAL_MACHINE` requires administrator privileges. In addition,
managing Hyper-V virtual machines (creating, starting, stopping them) requires
the current user to be a member of the **Hyper-V Administrators** group.

Macadam needs three kinds of vsock entries per machine:

| Purpose        | Description                                      | Default count |
|----------------|--------------------------------------------------|:------------:|
| **Network**    | User-mode networking between host and VM         | 1            |
| **Events**     | Notifications from the VM to the host (e.g. ready signal) | 1   |
| **Fileserver** | 9p file sharing — one entry per mount point      | 2            |

Each machine reserves its own set of entries from the available pool, so two
machines never share the same entry simultaneously.

## Two Ways to Set Up the Host

### Option A — Pre-allocate entries with `hyperv-prep` (recommended)

An administrator runs `hyperv-prep` once. This creates a pool of **persistent**
vsock registry entries and adds the current user to the Hyper-V Administrators
group. After that, the user can manage machines without ever needing
administrator rights again — including creating and deleting them.

```bash
# Run in an elevated (Administrator) terminal
macadam system hyperv-prep
```

The entries created this way are marked as **persistent**
(`KeepAfterMachineRemove`). They survive machine deletion and remain available
for future machines.

### Option B — On-demand elevation during `init`

If no pre-allocated entries exist when `macadam init` is executed, Macadam
detects that the registry is missing the required vsock entries. If the current
user belongs to the Windows *Administrators* group, a UAC (User Account Control)
prompt is shown to re-run the command with elevated privileges. The elevated
process creates the entries and, if needed, adds the user to the Hyper-V
Administrators group.

Entries created through on-demand elevation are **not** persistent — they are
tied to the machine lifecycle. When you remove the **last** machine that uses
non-persistent entries, the registry must be cleaned up, which again requires
administrator privileges. Macadam will show a UAC prompt at that point as well.

### Comparison

| Aspect | `hyperv-prep` (Option A) | On-demand elevation (Option B) |
|--------|--------------------------|-------------------------------|
| Registry entries persist after `rm` | Yes | No |
| Admin needed to create a machine | No (after initial prep) | Yes (UAC prompt) |
| Admin needed to remove a machine | No | Yes, when removing the last machine |
| Recommended for | Regular use, multi-user setups | Quick one-off experiments |

## `macadam system hyperv-prep` Reference

### Default mode (prepare)

```bash
macadam system hyperv-prep [flags]
```

Requires an **elevated terminal**. Creates vsock registry entries and adds the
current user to the Hyper-V Administrators group.

**Flags:**

- `--mounts <n>` — Number of Fileserver (mount) entries to create. Default: `2`.
- `--network <n>` — Number of Network entries to create. Default: `1`.
- `--events <n>` — Number of Events entries to create. Default: `1`.

If you plan to run multiple machines simultaneously or use many mount points,
increase these values accordingly. Each machine needs at least 1 Network entry,
1 Events entry, and 1 Fileserver entry per mount.

**Example — prepare enough entries for two machines with three mounts each:**

```bash
macadam system hyperv-prep --network 2 --events 2 --mounts 6
```

### `--status`

```bash
macadam system hyperv-prep --status
```

Does **not** require administrator privileges. Displays the current vsock
registry entries and whether the user is a member of the Hyper-V Administrators
group. Useful for diagnosing permission issues.

### `--reset`

```bash
macadam system hyperv-prep --reset [--force]
```

Requires an **elevated terminal**. Removes all vsock registry entries (including
persistent ones) and optionally removes the current user from the Hyper-V
Administrators group. A confirmation prompt is shown unless `--force` is passed.

## Machine Lifecycle and Permissions

### Creating a machine (`macadam init`)

1. Macadam checks whether enough vsock entries are available in the pool
   (entries not already assigned to another machine).
2. If enough entries exist **and** the user is a Hyper-V Administrator, the
   machine is created without elevation.
3. If entries are missing:
   - The user is prompted via UAC to elevate. The elevated process creates the
     missing entries (non-persistent) and adds the user to the Hyper-V
     Administrators group if needed.
   - If the user is not in the Windows Administrators group and cannot elevate,
     the command fails with an error suggesting `macadam system hyperv-prep`.

### Starting and stopping a machine (`macadam start` / `macadam stop`)

No administrator privileges are required as long as the user is a member of the
Hyper-V Administrators group.

### Removing a machine (`macadam rm`)

- If **persistent** entries were used (created via `hyperv-prep`), removing any
  machine — including the last one — does **not** require elevation, because the
  registry entries are kept for future use.
- If **non-persistent** entries were used (created via on-demand elevation) and
  this is the **last** machine, Macadam needs to clean up the registry entries,
  which requires administrator privileges. A UAC prompt will be shown.
- If other machines still exist, the registry entries are left in place
  regardless of their persistence flag, so no elevation is needed.

## Entry Assignment

Vsock entries form a shared pool, identified by their **purpose** (Network,
Events, Fileserver) and the **tool name** (macadam). When a new machine is
created, Macadam picks available entries from the pool — entries whose ports are
not already referenced by another machine's configuration. This prevents any
two machines from using the same vsock port.

The `--status` flag shows all entries and can help verify that the pool is large
enough for the number of machines you plan to run.

## Quick Start

```bash
# 1. Open an elevated (Administrator) PowerShell or Terminal

# 2. Prepare the host (one-time)
macadam system hyperv-prep

# 3. Switch to a normal (non-elevated) terminal

# 4. Create and start a machine
macadam init --provider hyperv fedora-cloud.vhdx
macadam start

# 5. Use the machine
macadam ssh

# 6. Clean up — no elevation needed
macadam rm --force
```

## Troubleshooting

| Symptom | Cause | Resolution |
|---------|-------|------------|
| *"insufficient HVSock registry entries for this VM"* | No available vsock entries in the pool | Run `macadam system hyperv-prep` in an elevated terminal, or increase the entry count with `--mounts`/`--network`/`--events` |
| *"Hyper-V machines require Hyper-V admin rights"* | User is not in the Hyper-V Administrators group | Run `macadam system hyperv-prep` or manually add the user to the group; log out and back in for the membership to take effect |
| *"removing this Hyper-V machine requires admin rights"* | Last machine uses non-persistent entries | Accept the UAC prompt, or pre-allocate persistent entries with `hyperv-prep` in the future |
| `hyperv-prep --status` shows entries but `init` still fails | Not enough **free** entries (others are assigned to existing machines) | Increase the pool: `macadam system hyperv-prep --mounts N --network N --events N` |
| Group membership changed but commands still fail | Windows token not refreshed | Log out and log back in for the new group membership to take effect |
