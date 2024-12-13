package macadam

import (
	crcos "github.com/crc-org/crc/v2/pkg/os"
	"golang.org/x/sys/unix"
)

func copyFile(src, dst string) error {
	if err := unix.Clonefile(src, dst, 0); err != nil {
		// fall back to a regular sparse copy, the error may be caused by the filesystem not supporting unix.CloneFile
		if err := crcos.CopyFileSparse(src, dst); err != nil {
			return err
		}
	}

	return nil
}
