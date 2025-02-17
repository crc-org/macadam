package imagepullers

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/podman/v5/pkg/machine/define"

	"github.com/containers/podman/v5/pkg/machine/env"
)

type NoopImagePuller struct {
	localPath   *define.VMFile
	sourceURI   string
	vmType      define.VMType
	machineName string
}

func NewNoopImagePuller(machineName string, vmType define.VMType) *NoopImagePuller {
	return &NoopImagePuller{
		machineName: machineName,
		vmType:      vmType,
	}
}

func (puller *NoopImagePuller) SetSourceURI(sourcePath string) {
	puller.sourceURI = sourcePath
}

func imageExtension(sourceURI string) string {
	if strings.HasSuffix(sourceURI, ".tar.gz") {
		return "tar.gz"
	}
	return filepath.Ext(sourceURI)
}

func (puller *NoopImagePuller) LocalPath() (*define.VMFile, error) {
	// if localPath has already been calculated returns it
	if puller.localPath != nil {
		return puller.localPath, nil
	}

	// calculate and set localPath
	dirs, err := env.GetMachineDirs(puller.vmType)
	if err != nil {
		return nil, err
	}

	vmFile, err := dirs.DataDir.AppendToNewVMFile(fmt.Sprintf("%s-%s.%s", puller.machineName, puller.vmType.String(), imageExtension(puller.sourceURI)), nil)
	if err != nil {
		return nil, err
	}
	puller.localPath = vmFile
	return vmFile, nil
}

/*
The noopImageBuilder does not actually download any image as the image is already stored locally.
The download func is used to make a copy of the source image so that the user image is not modified
by macadam
*/
func (puller *NoopImagePuller) Download() error {
	localPath, err := puller.LocalPath()
	if err != nil {
		return err
	}
	return copyFile(puller.sourceURI, localPath.Path)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	bufferedWriter := bufio.NewWriter(out)
	defer bufferedWriter.Flush()

	_, err = io.Copy(bufferedWriter, in)
	return err
}
