package osprovider

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

type CentosProvider struct {
	diskImage string
}

func NewCentosProvider() *CentosProvider {
	return &CentosProvider{}
}

func (centos *CentosProvider) Fetch(destDir string) (string, error) {
	arch := kernelArch()
	centosURL := fmt.Sprintf("https://cloud.centos.org/centos/10-stream/%s/images/CentOS-Stream-GenericCloud-10-latest.%s.qcow2", arch, arch)
	log.Infof("downloading %s to %s", centosURL, destDir)
	file, err := downloadOS(destDir, centosURL)
	if err != nil {
		return "", err
	}

	centos.diskImage = file

	return file, nil
}
