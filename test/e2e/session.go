package e2e

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	. "github.com/onsi/gomega"       //nolint:staticcheck
	. "github.com/onsi/gomega/gexec" //nolint:staticcheck
)

var DefaultWaitTimeout = 90

// MacadamSession wraps the gexec.session so we can extend it
type MacadamSession struct {
	*Session
}

// WaitWithDefaultTimeout waits for process finished with DefaultWaitTimeout
func (s *MacadamSession) WaitWithDefaultTimeout() {
	s.WaitWithTimeout(DefaultWaitTimeout)
}

// WaitWithTimeout waits for process finished with DefaultWaitTimeout
func (s *MacadamSession) WaitWithTimeout(timeout int) {
	Eventually(s, timeout).Should(Exit(), func() string {
		// Note eventually does not kill the command as such the command is leaked forever without killing it
		// Also let's use SIGABRT to create a go stack trace so in case there is a deadlock we see it.
		s.Signal(syscall.SIGABRT)
		// Give some time to let the command print the output so it is not printed much later
		// in the log at the wrong place.
		time.Sleep(1 * time.Second)
		// As the output is logged by default there no need to dump it here.
		return fmt.Sprintf("command timed out after %ds: %v",
			timeout, s.Command.Args)
	})
	os.Stdout.Sync()
	os.Stderr.Sync()
}

// OutputToString formats session output to string
func (s *MacadamSession) OutputToString() string {
	if s == nil || s.Out == nil || s.Out.Contents() == nil {
		return ""
	}

	fields := strings.Fields(string(s.Out.Contents()))
	return strings.Join(fields, " ")
}

// ErrorToString formats session stderr to string
func (s *MacadamSession) ErrorToString() string {
	fields := strings.Fields(string(s.Err.Contents()))
	return strings.Join(fields, " ")
}
