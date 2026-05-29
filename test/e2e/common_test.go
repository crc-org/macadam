package e2e

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/crc-org/macadam/pkg/imagepullers"
	"go.podman.io/podman/v6/pkg/machine/define"

	"github.com/crc-org/macadam/test/osprovider"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegaformat "github.com/onsi/gomega/format"
	"github.com/onsi/gomega/gexec"
	"github.com/spf13/pflag"
)

var (
	macadamTest      *MacadamTestIntegration
	machineResponses []ListReporter
	err              error
	tempDir          string
	IMAGE            string
	MACADAM_BINARY   string
	keypath          string
	cloudinitPath    string
)

func TestMain(m *testing.M) {
	RegisterFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	os.Exit(m.Run())
}

func RegisterFlags(flags *flag.FlagSet) {
	flags.StringVar(&IMAGE, "image", "", "Path to the image used in tests.")
	flags.StringVar(&MACADAM_BINARY, "macadam-binary", "", "Path to the macadam binary to be tested.")
}

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)

	// disable error/output strings truncation
	gomegaformat.MaxLength = 0

	// fetch the current (reporter) config
	suiteConfig, reporterConfig := GinkgoConfiguration()

	err := os.MkdirAll("out", 0775)
	if err != nil {
		GinkgoWriter.Printf("failed to create directory: %v", err)
	}
	reporterConfig.JUnitReport = filepath.Join("out", "e2e.xml")

	RunSpecs(t, "Macadam suite", suiteConfig, reporterConfig)
}

func noVMcheck(opts ...string) {
	listCMD := []string{"list", "--format", "json"}
	if len(opts) > 0 {
		listCMD = append(listCMD, "--provider", opts[0])
	}

	session := macadamTest.Macadam(listCMD)
	session.WaitWithDefaultTimeout()
	Expect(session).Should(gexec.Exit(0))
	err := json.Unmarshal(session.Out.Contents(), &machineResponses)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(machineResponses)).Should(Equal(0))
}

func vmNumberCheck(expect_number int, opts ...string) {
	listCMD := []string{"list", "--format", "json"}
	if len(opts) > 0 {
		listCMD = append(listCMD, "--provider", opts[0])
	}

	session := macadamTest.Macadam(listCMD)
	session.WaitWithDefaultTimeout()
	Expect(session).Should(gexec.Exit(0))
	err := json.Unmarshal(session.Out.Contents(), &machineResponses)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(machineResponses)).Should(Equal(expect_number))
}

func runCMDsuccess(cmd []string) string {
	GinkgoWriter.Printf("Run cmd: %v\n", cmd)
	session := macadamTest.Macadam(cmd)
	session.WaitWithDefaultTimeout()
	Expect(session).Should(gexec.Exit(0))
	return session.OutputToString()
}

func runCMD(cmd []string) (int, string, string) {
	GinkgoWriter.Printf("Run cmd: %v\n", cmd)
	session := macadamTest.Macadam(cmd)
	session.WaitWithDefaultTimeout()
	return session.ExitCode(), session.OutputToString(), session.ErrorToString()
}

var DefaultNoWaitTimeout = 2 * time.Second

// Run command with short timeout, then terminate it. Used for commands that hang
// and never exit. (e.g., SSH with -t in non-interactive environments like containers
// or CI/CD pipelines)
// We wait 1 second for output, then sends SIGABRT. This works for both
// podman run -it and podman run -d scenarios.
func runCMDNoWait(cmd []string) string {
	GinkgoWriter.Printf("Run ssh cmd: %v\n", cmd)
	session := macadamTest.Macadam(cmd)
	time.Sleep(DefaultNoWaitTimeout)
	session.Signal(syscall.SIGABRT)
	os.Stdout.Sync()
	return session.OutputToString()
}

func removeAllVM(opts ...string) {
	listCMD := []string{"list", "--format", "json"}
	baseCMD := []string{}
	if len(opts) > 0 {
		provider := opts[0]
		listCMD = append(listCMD, "--provider", provider)
		baseCMD = append(baseCMD, "--provider", provider)
	}

	session := macadamTest.Macadam(listCMD)
	session.WaitWithDefaultTimeout()
	Expect(session).Should(gexec.Exit(0))
	err = json.Unmarshal(session.Out.Contents(), &machineResponses)
	Expect(err).NotTo(HaveOccurred())

	for _, m := range machineResponses {
		stopCMD := append([]string{"stop"}, baseCMD...)
		stopCMD = append(stopCMD, m.Name)
		rmCMD := append([]string{"rm", "-f"}, baseCMD...)
		rmCMD = append(rmCMD, m.Name)

		// stop the VM
		session = macadamTest.Macadam(stopCMD)
		session.WaitWithDefaultTimeout()
		Expect(session).Should(gexec.Exit(0))
		Expect(session.OutputToString()).Should(ContainSubstring("stopped successfully"))

		// rm the VM and verify that "list" does not return any vm
		session = macadamTest.Macadam(rmCMD)
		session.WaitWithDefaultTimeout()
		Expect(session).Should(gexec.Exit(0))
	}

	session = macadamTest.Macadam(listCMD)
	session.WaitWithDefaultTimeout()
	Expect(session).Should(gexec.Exit(0))
	err = json.Unmarshal(session.Out.Contents(), &machineResponses)
	Expect(err).NotTo(HaveOccurred())
	Expect(len(machineResponses)).Should(Equal(0))
}

var _ = BeforeSuite(func() {
	tempDir, err = os.MkdirTemp("", "test-")
	Expect(err).NotTo(HaveOccurred())

	if IMAGE != "" {
		if runtime.GOOS == "windows" {
			_, err = imagepullers.ImageExtension(define.HyperVVirt, IMAGE)
			if err != nil {
				_, err = imagepullers.ImageExtension(define.WSLVirt, IMAGE)
			}
		} else {
			_, err = imagepullers.ImageExtension(define.UnknownVirt, IMAGE)
		}
		Expect(err).NotTo(HaveOccurred())
	} else {
		if runtime.GOOS != "windows" {
			// download CentOS image
			centosProvider := osprovider.NewCentosProvider()
			IMAGE, err = centosProvider.Fetch(tempDir)
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("Using downloaded centos qcow2 image %s for %s test", IMAGE, runtime.GOOS)
		} else {
			fmt.Printf("Please provide a VHD/VHDX or tar.gz image using --image flag for Windows tests.")
			os.Exit(1)
		}
	}

	keypath = filepath.Join(tempDir, "id_rsa")
	cloudinitPath = filepath.Join(tempDir, "user-data")
	//generate ssh key
	cmd := exec.Command("ssh-keygen", "-t", "rsa", "-f", keypath, "-N", "")
	err = cmd.Run()
	Expect(err).ShouldNot(HaveOccurred())
	//copy user-data
	wd, err := os.Getwd()
	Expect(err).ShouldNot(HaveOccurred())
	cloudinit := wd + "/../testdata/user-data"
	content, err := os.ReadFile(cloudinit)
	Expect(err).ShouldNot(HaveOccurred())
	//replace ssh pub key to user-data file
	pubkeypath := keypath + ".pub"
	pubkey, err := os.ReadFile(pubkeypath)
	Expect(err).ShouldNot(HaveOccurred())
	newContent := strings.ReplaceAll(string(content), "[sshkey]", string(pubkey))
	err = os.WriteFile(cloudinitPath, []byte(newContent), 0644)
	Expect(err).ShouldNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	os.RemoveAll(tempDir)
})

var _ = BeforeEach(func() {
	macadamTest = MacadamTestCreate()
})
