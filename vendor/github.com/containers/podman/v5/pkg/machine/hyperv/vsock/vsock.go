//go:build windows

package vsock

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/Microsoft/go-winio"
	"github.com/containers/podman/v5/pkg/machine/env"
	"github.com/containers/podman/v5/pkg/machine/sockets"
	"github.com/containers/podman/v5/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/registry"
)

var (
	ErrVSockRegistryEntryExists = errors.New("registry entry already exists")
	// ErrNoReusableHVSockRegistryEntry means no pre-created hvsock registry
	// entry matches the machine and purpose.
	ErrNoReusableHVSockRegistryEntry = errors.New("no reusable hvsock registry entry found")
)

const (
	// HvsockMachineName is the string identifier for the machine name in a registry entry
	HvsockMachineName = "MachineName"
	// HvsockToolName is the string identifier for the tool name in a registry entry
	HvsockToolName = "ToolName"
	// HvsockPurpose is the string identifier for the sock purpose in a registry entry
	HvsockPurpose = "Purpose"
	// HvsockKeepAfterMachineRemove is a flag to avoid that the entry gets deleted when
	// the machine using it gets deleted. This is important for use cases where an
	// Administrator pre-creates the registry key so that users can create and delete
	// machines without requiring admin privileges.
	HvsockKeepAfterMachineRemove = "KeepAfterMachineRemove"
	// VsockRegistryPath describes the registry path to where the hvsock registry entries live
	VsockRegistryPath = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\Virtualization\GuestCommunicationServices`
	// LinuxVM is the default guid for a Linux VM on Windows
	LinuxVM = "FACB-11E6-BD58-64006A7986D3"
)

// HVSockPurpose describes what the hvsock is needed for
type HVSockPurpose int

const (
	// Network implies the sock is used for user-mode networking
	Network HVSockPurpose = iota
	// Events implies the sock is used for notification (like "Ready")
	Events
	// Fileserver implies that the sock is used for serving files from host to VM
	Fileserver
)

func (hv HVSockPurpose) string() string {
	switch hv {
	case Network:
		return "Network"
	case Events:
		return "Events"
	case Fileserver:
		return "Fileserver"
	}
	return ""
}

func (hv HVSockPurpose) Equal(purpose string) bool {
	return hv.string() == purpose
}

func toHVSockPurpose(p string) (HVSockPurpose, error) {
	switch p {
	case "Network":
		return Network, nil
	case "Events":
		return Events, nil
	case "Fileserver":
		return Fileserver, nil
	}
	return 0, fmt.Errorf("unknown hvsockpurpose: %s", p)
}

func openVSockRegistryEntry(entry string) (registry.Key, error) {
	return registry.OpenKey(registry.LOCAL_MACHINE, entry, registry.QUERY_VALUE)
}

// HVSockRegistryEntry describes a registry entry used in Windows for HVSOCK implementations
type HVSockRegistryEntry struct {
	KeyName     string        `json:"key_name"`
	Purpose     HVSockPurpose `json:"purpose"`
	Port        uint64        `json:"port"`
	MachineName string        `json:"machineName"`

	// ToolName identifies the application that created this registry entry (e.g., Podman).
	// This provides information about the entry's origin and can be used for filter entries
	// if purpose is not enough.
	ToolName string `json:"creator_tool,omitempty"`

	// KeepAfterMachineRemove is set to true if the registry entry should not be deleted
	// when the user deletes or reset the Podman machines.
	// This is useful to let unprivileged users create and delete machines without
	// requiring privileges to edit the registry.
	KeepAfterMachineRemove bool `json:"keep_after_machine_remove,omitempty"`
}

// Add creates a new Windows registry entry with string values from the
// HVSockRegistryEntry.
func (hv *HVSockRegistryEntry) Add() error {
	if err := hv.validate(); err != nil {
		return err
	}
	exists, err := hv.exists()
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("%w: %s", ErrVSockRegistryEntryExists, hv.KeyName)
	}
	parentKey, err := registry.OpenKey(registry.LOCAL_MACHINE, VsockRegistryPath, registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer func() {
		if err := parentKey.Close(); err != nil {
			logrus.Error(err)
		}
	}()
	newKey, openedExisting, err := registry.CreateKey(parentKey, hv.KeyName, registry.WRITE)
	if err != nil {
		return err
	}
	defer func() {
		if err := newKey.Close(); err != nil {
			logrus.Error(err)
		}
	}()
	// The key may have been created by someone else after the exists() check;
	// never overwrite the values of a key we did not create.
	if openedExisting {
		return fmt.Errorf("%w: %s", ErrVSockRegistryEntryExists, hv.KeyName)
	}

	if err := newKey.SetStringValue(HvsockPurpose, hv.Purpose.string()); err != nil {
		return err
	}
	if err := newKey.SetStringValue(HvsockMachineName, hv.MachineName); err != nil {
		return err
	}
	if err := newKey.SetStringValue(HvsockToolName, hv.ToolName); err != nil {
		return err
	}
	// Save KeepAfterMachineRemove flag if set
	if hv.KeepAfterMachineRemove {
		var value uint32 = 1
		if err := newKey.SetDWordValue(HvsockKeepAfterMachineRemove, value); err != nil {
			return err
		}
	}
	return nil
}

// Remove deletes the registry key and its string values. Entries marked with
// KeepAfterMachineRemove (e.g. pre-created by a privileged helper tool) are
// never deleted so they can be reused by future machines.
func (hv *HVSockRegistryEntry) Remove() error {
	if hv.KeepAfterMachineRemove {
		logrus.Debugf("keeping hvsock registry entry %s: KeepAfterMachineRemove is set", hv.KeyName)
		return nil
	}
	return registry.DeleteKey(registry.LOCAL_MACHINE, hv.fqPath())
}

func (hv *HVSockRegistryEntry) fqPath() string {
	return fmt.Sprintf("%s\\%s", VsockRegistryPath, hv.KeyName)
}

func (hv *HVSockRegistryEntry) validate() error {
	if hv.Port < 1 {
		return errors.New("port must be larger than 1")
	}
	if len(hv.Purpose.string()) < 1 {
		return errors.New("required field purpose is empty")
	}
	if len(hv.MachineName) < 1 {
		return errors.New("required field machinename is empty")
	}
	if len(hv.ToolName) < 1 {
		return errors.New("required field toolName is empty")
	}
	if len(hv.KeyName) < 1 {
		return errors.New("required field keypath is empty")
	}
	return nil
}

func (hv *HVSockRegistryEntry) exists() (bool, error) {
	foo := hv.fqPath()
	_ = foo
	k, err := openVSockRegistryEntry(hv.fqPath())
	if err == nil {
		if err := k.Close(); err != nil {
			logrus.Errorf("failed to close registry key: %v", err)
		}
		return true, nil
	}
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// findOpenHVSockPort looks for an open random port. it verifies the port is not
// already being used by another hvsock in the Windows registry.
func findOpenHVSockPort() (uint64, error) {
	// If we cannot find a free port in 10 attempts, something is wrong
	for i := 0; i < 10; i++ {
		port, err := utils.GetRandomPort()
		if err != nil {
			return 0, err
		}
		// Try and load registry entries by port to see if they exist
		_, err = LoadHVSockRegistryEntry(uint64(port))
		if err == nil {
			// the port is no good, it is being used; try again
			logrus.Errorf("port %d is already used for hvsock", port)
			continue
		}
		if errors.Is(err, registry.ErrNotExist) {
			// the port is good to go
			return uint64(port), nil
		}
		if err != nil {
			// something went wrong
			return 0, err
		}
	}
	return 0, errors.New("unable to find a free port for hvsock use")
}

// NewHVSockRegistryEntry is a constructor to make a new registry entry in Windows.  After making the new
// object, you must call the add() method to *actually* add it to the Windows registry.
func NewHVSockRegistryEntry(machineName string, purpose HVSockPurpose) (*HVSockRegistryEntry, error) {
	// a so-called wildcard entry ... everything from FACB -> 6D3 is MS special sauce
	// for a " linux vm".  this first segment is hexi for the hvsock port number
	// 00000400-FACB-11E6-BD58-64006A7986D3
	port, err := findOpenHVSockPort()
	if err != nil {
		return nil, err
	}
	r := HVSockRegistryEntry{
		KeyName:     PortToKeyName(port),
		Purpose:     purpose,
		Port:        port,
		MachineName: machineName,
		ToolName:    env.GetToolName(),
		// entries created here are owned by the machine and removed with it;
		// pre-created reusable entries are loaded from the registry instead
		KeepAfterMachineRemove: false,
	}
	if err := r.Add(); err != nil {
		return nil, err
	}
	return &r, nil
}

func PortToKeyName(port uint64) string {
	// this could be flattened but given the complexity, I thought it might
	// be more difficult to read
	hexi := strings.ToUpper(fmt.Sprintf("%08x", port))
	return fmt.Sprintf("%s-%s", hexi, LinuxVM)
}

func parseHexToUint64(hex string) (uint64, error) {
	return strconv.ParseUint(hex, 16, 64)
}

// portFromKeyName extracts the hvsock port number from a registry key name as
// produced by PortToKeyName ("<PORTHEX>-FACB-11E6-BD58-64006A7986D3").
func portFromKeyName(keyName string) (uint64, error) {
	prefix, suffix, found := strings.Cut(keyName, "-")
	if !found || !strings.EqualFold(suffix, LinuxVM) {
		return 0, fmt.Errorf("malformed key name %q: expected %q suffix", keyName, LinuxVM)
	}
	if len(prefix) < 1 || len(prefix) > 8 {
		return 0, fmt.Errorf("malformed key name %q: invalid port segment %q", keyName, prefix)
	}
	port, err := parseHexToUint64(prefix)
	if err != nil {
		return 0, fmt.Errorf("malformed key name %q: %w", keyName, err)
	}
	return port, nil
}

func LoadHVSockRegistryEntry(port uint64) (*HVSockRegistryEntry, error) {
	return loadHVSockRegistryEntryFromSubKey(PortToKeyName(port))
}

// Listener returns a net.Listener for the given HvSock.
func (hv *HVSockRegistryEntry) Listener() (net.Listener, error) {
	n := winio.HvsockAddr{
		VMID:      winio.HvsockGUIDWildcard(), // When listening on the host side, use equiv of 0.0.0.0
		ServiceID: winio.VsockServiceID(uint32(hv.Port)),
	}
	listener, err := winio.ListenHvsock(&n)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

// ListenSetupWait creates an hvsock on the windows side and returns
// a wait function that, when called, blocks until it receives a ready
// notification on the vsock
func (hv *HVSockRegistryEntry) ListenSetupWait() (func() error, io.Closer, error) {
	listener, err := hv.Listener()
	if err != nil {
		return nil, nil, err
	}

	errChan := make(chan error)
	go sockets.ListenAndWaitOnSocket(errChan, listener)
	return func() error {
		return <-errChan
	}, listener, nil
}

// loadHVSockRegistryEntries returns the pre-created hvsock registry entries the
// given machine can reuse, filtered by purpose, and optionally limited by size.
// If limit is -1, it returns all matching entries. Otherwise, it returns up to
// 'limit' entries.
func loadHVSockRegistryEntries(machineName string, purpose HVSockPurpose, limit int) ([]*HVSockRegistryEntry, error) {
	parentKey, err := registry.OpenKey(registry.LOCAL_MACHINE, VsockRegistryPath, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := parentKey.Close(); err != nil {
			logrus.Errorf("failed to close registry key: %v", err)
		}
	}()

	subKeyNames, err := parentKey.ReadSubKeyNames(-1)
	if err != nil {
		return nil, err
	}

	allEntries := []*HVSockRegistryEntry{}
	for _, subKeyName := range subKeyNames {
		entry, err := loadHVSockRegistryEntryFromSubKey(subKeyName)
		if err != nil {
			// skip malformed/unreadable keys
			logrus.Debugf("skipping hvsock registry entry %s: %v", subKeyName, err)
			continue
		}
		if !entryCanBeReusedBy(entry, machineName, purpose) {
			continue
		}
		allEntries = append(allEntries, entry)
	}

	// sort by port so the choice of entries is predictable
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Port < allEntries[j].Port
	})
	if limit != -1 && len(allEntries) > limit {
		allEntries = allEntries[:limit]
	}
	return allEntries, nil
}

// loadHVSockRegistryEntryFromSubKey loads a single hvsock registry entry
// described by its subkey name. It returns an error if the subkey cannot be
// read or is malformed.
func loadHVSockRegistryEntryFromSubKey(subKeyName string) (*HVSockRegistryEntry, error) {
	fqPath := fmt.Sprintf("%s\\%s", VsockRegistryPath, subKeyName)
	k, err := openVSockRegistryEntry(fqPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := k.Close(); err != nil {
			logrus.Errorf("failed to close registry key: %v", err)
		}
	}()

	p, _, err := k.GetStringValue(HvsockPurpose)
	if err != nil {
		return nil, err
	}
	purpose, err := toHVSockPurpose(p)
	if err != nil {
		return nil, err
	}

	machineName, _, err := k.GetStringValue(HvsockMachineName)
	if err != nil {
		return nil, err
	}

	// ToolName is informational and optional.
	toolName, _, err := k.GetStringValue(HvsockToolName)
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return nil, err
	}

	keep := readKeepAfterMachineRemove(k, subKeyName)

	port, err := portFromKeyName(subKeyName)
	if err != nil {
		return nil, err
	}

	return &HVSockRegistryEntry{
		KeyName:                subKeyName,
		Purpose:                purpose,
		Port:                   port,
		MachineName:            machineName,
		ToolName:               toolName,
		KeepAfterMachineRemove: keep,
	}, nil
}

// readKeepAfterMachineRemove reads the optional KeepAfterMachineRemove value of
// an hvsock registry key. The expected type is REG_DWORD, but REG_QWORD and
// boolean-like REG_SZ values ("1", "true", ...) written by external tools are
// accepted too. A missing or unparsable value means false; an unparsable value
// is logged so the entry is not silently ignored.
func readKeepAfterMachineRemove(k registry.Key, keyName string) bool {
	value, _, err := k.GetIntegerValue(HvsockKeepAfterMachineRemove)
	if err == nil {
		return value != 0
	}
	if errors.Is(err, registry.ErrNotExist) {
		return false
	}
	if errors.Is(err, registry.ErrUnexpectedType) {
		if s, _, serr := k.GetStringValue(HvsockKeepAfterMachineRemove); serr == nil {
			if keep, perr := strconv.ParseBool(strings.TrimSpace(s)); perr == nil {
				return keep
			}
		}
	}
	logrus.Warnf("ignoring invalid %s value on hvsock registry entry %s (expected REG_DWORD 0 or 1): %v",
		HvsockKeepAfterMachineRemove, keyName, err)
	return false
}

// entryCanBeReusedBy reports whether the given registry entry can be reused by
// the machine named machineName for the given purpose. Only entries pre-created
// for reuse (KeepAfterMachineRemove set) match. Each entry is bound to a single
// machine through its MachineName: machines can run at the same time, so two
// machines must never share a port.
func entryCanBeReusedBy(entry *HVSockRegistryEntry, machineName string, purpose HVSockPurpose) bool {
	return entry.KeepAfterMachineRemove &&
		entry.Purpose == purpose &&
		entry.MachineName == machineName
}

// LoadHVSockRegistryEntryByPurpose returns a single pre-created hvsock registry
// entry that the given machine can reuse for the given purpose. It returns an
// error wrapping ErrNoReusableHVSockRegistryEntry if no reusable entry exists.
func LoadHVSockRegistryEntryByPurpose(machineName string, purpose HVSockPurpose) (*HVSockRegistryEntry, error) {
	entries, err := loadHVSockRegistryEntries(machineName, purpose, 1)
	if err != nil {
		return nil, err
	}
	if len(entries) != 1 {
		return nil, fmt.Errorf("%w for machine %s and purpose %s", ErrNoReusableHVSockRegistryEntry, machineName, purpose.string())
	}
	return entries[0], nil
}

// LoadOrCreateHVSockRegistryEntry returns a pre-created hvsock registry entry
// the given machine can reuse for the given purpose. If there is none, or the
// registry cannot be searched, it creates a new entry owned by the machine.
// Creating an entry requires write access to HKLM (admin rights).
func LoadOrCreateHVSockRegistryEntry(machineName string, purpose HVSockPurpose) (*HVSockRegistryEntry, error) {
	entry, err := LoadHVSockRegistryEntryByPurpose(machineName, purpose)
	if err == nil {
		logrus.Debugf("Reusing pre-created %s hvsock registry entry %s for machine %s", purpose.string(), entry.KeyName, machineName)
		return entry, nil
	}
	logLookupFailure(machineName, purpose, err)
	return NewHVSockRegistryEntry(machineName, purpose)
}

// LoadReusableHVSockRegistryEntries returns all pre-created hvsock registry
// entries the given machine can reuse for the given purpose. A failed registry
// search is logged and treated as "no entries" so callers can fall back to
// creating new ones.
func LoadReusableHVSockRegistryEntries(machineName string, purpose HVSockPurpose) []*HVSockRegistryEntry {
	entries, err := LoadAllHVSockRegistryEntriesByPurpose(machineName, purpose)
	if err != nil {
		logLookupFailure(machineName, purpose, err)
		return nil
	}
	return entries
}

// logLookupFailure logs why no pre-created entry is used: debug when there is
// simply none, warning when the registry search itself failed.
func logLookupFailure(machineName string, purpose HVSockPurpose, err error) {
	if errors.Is(err, ErrNoReusableHVSockRegistryEntry) {
		logrus.Debugf("No reusable %s hvsock registry entry for machine %s, creating a new one", purpose.string(), machineName)
		return
	}
	logrus.Warnf("Unable to look up reusable %s hvsock registry entries for machine %s, creating new ones: %v", purpose.string(), machineName, err)
}

// LoadAllHVSockRegistryEntriesByPurpose returns all pre-created hvsock registry
// entries that the given machine can reuse for the given purpose.
func LoadAllHVSockRegistryEntriesByPurpose(machineName string, purpose HVSockPurpose) ([]*HVSockRegistryEntry, error) {
	return loadHVSockRegistryEntries(machineName, purpose, -1)
}
