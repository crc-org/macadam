//go:build !darwin

package macadam

import (
	crcos "github.com/crc-org/crc/v2/pkg/os"
)

func copyFile(src, dst string) error {
	return crcos.CopyFile(src, dst)
}
