package macadam

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containers/podman/v5/pkg/machine/define"
	"github.com/containers/podman/v5/pkg/machine/vmconfigs"
)

type CrcImagePuller struct {
	localPath     *define.VMFile
	sourcePath    string
	vmType        define.VMType
	machineConfig *vmconfigs.MachineConfig
	machineDirs   *define.MachineDirs
}

func GetHomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("Failed to get homeDir: " + err.Error())
	}
	return homeDir
}

var (
	CrcBaseDir         = filepath.Join(GetHomeDir(), ".crc")
	MachineBaseDir     = CrcBaseDir
	MachineCacheDir    = filepath.Join(MachineBaseDir, "cache")
	MachineInstanceDir = filepath.Join(MachineBaseDir, "machines")
)

func NewCrcImagePuller(vmType define.VMType) (*CrcImagePuller, error) {
	crcImage := CrcImagePuller{
		vmType: vmType,
	}

	return &crcImage, nil
}

/* TODO: Might be better to pass an actual URI and to strip the "file://" part from it */
func (puller *CrcImagePuller) SetSourceURI(path string) {
	puller.sourcePath = path
}

/*
	type MachineDirs struct {
		ConfigDir     *VMFile
		DataDir       *VMFile
		ImageCacheDir *VMFile
		RuntimeDir    *VMFile
	}

var crcMachineDirs  = MachineDirs {
ConfigDir: CrcBaseDir,
DataDir: filepath.Join(MachineInstanceDir, , "crc"),
ImageCacheDir: MachineCacheDir,
RuntimeDir: filepath.Join(MachineInstanceDir, , "crc"),
}
*/

func imageExtension(vmType define.VMType) string {
	switch vmType {
	case define.QemuVirt:
		return ".qcow2"
	case define.AppleHvVirt, define.LibKrun:
		return ".raw"
	case define.HyperVVirt:
		return ".vhdx"
	case define.WSLVirt:
		return ""
	default:
		return ""
	}
}

func (puller *CrcImagePuller) LocalPath() (*define.VMFile, error) {
	// filename is bundle specific
	return define.NewMachineFile(filepath.Join(MachineInstanceDir, "crc", fmt.Sprintf("crc%s", imageExtension(puller.vmType))), nil)
}

func (puller *CrcImagePuller) Download() error {
	//_ = bundle.Get()
	// no download yet, reuse crc code
	imagePath, err := puller.LocalPath()
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("%+v", puller))
	slog.Info("file copy", "source", puller.sourcePath, "dest", imagePath.GetPath())
	if err := copyFile(puller.sourcePath, imagePath.GetPath()); err != nil {
		return err
	}
	/*
		if err := unix.Access(imagePath.GetPath(), unix.R_OK|unix.W_OK); err != nil {
			return fmt.Errorf("cannot access %s: %w", imagePath.GetPath(), err)
		}
	*/
	fi, err := os.Stat(imagePath.GetPath())
	if err != nil {
		return fmt.Errorf("cannot get file information for %s: %w", imagePath.GetPath(), err)
	}
	perms := fi.Mode().Perm()
	if !perms.IsRegular() {
		return fmt.Errorf("%s must be a regular file", imagePath.GetPath())
	}
	if (perms & 0600) != 0600 {
		return fmt.Errorf("%s is not readable/writable by the user", imagePath.GetPath())
	}
	slog.Info("all is fine", "imagePath", imagePath.GetPath())

	return nil
}

/*
	bundleInfo, err := bundle.Use(bundleName)
	if err == nil {
		logging.Infof("Loading bundle: %s...", bundleName)
		return bundleInfo, nil
	}
	logging.Debugf("Failed to load bundle %s: %v", bundleName, err)
	logging.Infof("Downloading bundle: %s...", bundleName)
	bundlePath, err = bundle.Download(preset, bundlePath, enableBundleQuayFallback)
	if err != nil {
		return nil, err
	}
	logging.Infof("Extracting bundle: %s...", bundleName)
	if _, err := bundle.Extract(bundlePath); err != nil {
		return nil, err
	}
	return bundle.Use(bundleName)
*/
