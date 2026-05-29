package imagepullers

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"go.podman.io/podman/v6/pkg/machine/define"

	"go.podman.io/podman/v6/pkg/machine/env"
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

	imageExt, err := ImageExtension(puller.vmType, puller.sourceURI)
	if err != nil {
		return nil, err
	}

	vmFile, err := dirs.DataDir.AppendToNewVMFile(fmt.Sprintf("%s-%s%s", puller.machineName, puller.vmType.String(), imageExt), nil)
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

	src, err := os.Open(puller.sourceURI)
	if err != nil {
		return err
	}
	defer src.Close()

	return doCopyFile(src, localPath.Path)
}

func copyFile(src *os.File, dest string) error {
	destF, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destF.Close()

	bufferedWriter := bufio.NewWriter(destF)
	defer bufferedWriter.Flush()

	_, err = io.Copy(bufferedWriter, src)
	return err
}
