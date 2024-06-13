package main

import (
	"fmt"
	"log/slog"

	"github.com/crc-org/macadam/pkg/cmdline"
)

func main() {
	slog.Info(fmt.Sprintf("macadam version %s", cmdline.Version()))
}
