package e2e

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type Init_Hardware_Parameter struct {
	cpu           string
	disk          string
	memory        string
	expect_memory []string
}
type Init_Test_Para []Init_Hardware_Parameter

func init_hardware_test(imagePath string, tests Init_Hardware_Parameter, opts ...string) {

	initCMD := []string{}
	startCMD := []string{"start"}
	checkCPU := []string{"ssh", "nproc"}
	checkDisk := []string{"ssh", "lsblk"}
	checkMemory := []string{"ssh", "free", "-h"}

	if len(opts) > 0 {
		provider := []string{"--provider", opts[0]}
		initCMD = append(initCMD, provider...)
		startCMD = append(startCMD, provider...)
		checkCPU = append(provider, checkCPU...)
		checkDisk = append(provider, checkDisk...)
		checkMemory = append(provider, checkMemory...)
	}

	initCMD = append(initCMD, "init", "--cpus", tests.cpu, "--disk-size", tests.disk, "--memory", tests.memory, imagePath)
	// init a  VM with cpu and disk-size setup
	runCMDsuccess(initCMD)

	// check the list command returns one item
	vmNumberCheck(1, opts...)

	// start the  VM
	output := runCMDsuccess(startCMD)
	Expect(output).Should(ContainSubstring("started successfully"))

	// ssh into the VM and check CPU number
	output = runCMDNoWait(checkCPU)
	Expect(strings.TrimSpace(output)).Should(Equal(tests.cpu))

	// ssh into the VM and check disk
	output = runCMDNoWait(checkDisk)
	Expect(output).Should(ContainSubstring(tests.disk))

	// ssh into the VM and check memory
	output = runCMDNoWait(checkMemory)
	// Verify memory is close to the set memory (allow for system overhead)
	Expect(output).Should(Or(ContainSubstring(tests.expect_memory[0]), ContainSubstring(tests.expect_memory[1]), ContainSubstring(tests.expect_memory[2]), ContainSubstring(tests.expect_memory[3])))
}

func init_sshkey_test(imagePath string, opts ...string) {
	initCMD := []string{"init", "--username", "test", "--ssh-identity-path", keypath, imagePath}
	startCMD := []string{"start"}
	usernameCheck := []string{"ssh", "--username", "test", "whoami"}

	if len(opts) > 0 {
		provider := []string{"--provider", opts[0]}
		initCMD = append(initCMD, provider...)
		startCMD = append(startCMD, provider...)
		usernameCheck = append(provider, usernameCheck...)
	}

	runCMDsuccess(initCMD)
	vmNumberCheck(1, opts...)

	output := runCMDsuccess(startCMD)
	Expect(output).Should(ContainSubstring("started successfully"))

	output = runCMDNoWait(usernameCheck)
	Expect(strings.TrimSpace(output)).Should(Equal("test"))
}

func init_vmName_test(imagePath string, opts ...string) {
	initCMD := []string{"init", "--name", "myVM", imagePath}
	startCMD := []string{"start", "myVM"}
	checkUser := []string{"ssh", "myVM", "whoami"}

	if len(opts) > 0 {
		provider := []string{"--provider", opts[0]}
		initCMD = append(initCMD, provider...)
		startCMD = append(startCMD, provider...)
		checkUser = append(provider, checkUser...)
	}

	runCMDsuccess(initCMD)

	// check the list command returns one item
	vmNumberCheck(1, opts...)

	// start the VM with set name
	output := runCMDsuccess(startCMD)
	Expect(output).Should(ContainSubstring("started successfully"))

	// ssh into the VM and prints user
	output = runCMDNoWait(checkUser)
	expectedUser := "core"
	if len(opts) > 0 && opts[0] == "wsl" {
		expectedUser = "user"
	}
	Expect(strings.TrimSpace(output)).Should(Equal(expectedUser))
}

func init_cloudinit_test(imagePath string, opts ...string) {
	initCMD := []string{"init", "--cloud-init", cloudinitPath, "--username", "macadamtest", "--ssh-identity-path", keypath, imagePath}
	startCMD := []string{"start"}
	checkuser := []string{"ssh", "whoami"}
	check_cloudFinal := []string{"ssh", "systemctl", "status", "cloud-final"}
	check_gitVersion := []string{"ssh", "git", "--version"}
	check_file_exist := []string{"ssh", "cat", "/home/macadamtest/hello.txt"}

	if len(opts) > 0 {
		provider := []string{"--provider", opts[0]}
		initCMD = append(initCMD, provider...)
		startCMD = append(startCMD, provider...)
		checkuser = append(provider, checkuser...)
		check_cloudFinal = append(provider, check_cloudFinal...)
		check_gitVersion = append(provider, check_gitVersion...)
		check_file_exist = append(provider, check_file_exist...)
	}

	runCMDsuccess(initCMD)

	// check the list command returns one item
	vmNumberCheck(1, opts...)

	// start the VM
	output := runCMDsuccess(startCMD)
	Expect(output).Should(ContainSubstring("started successfully"))

	// ssh into the VM and prints user
	output = runCMDNoWait(checkuser)
	Expect(strings.TrimSpace(output)).Should(Equal("macadamtest"))

	// wait until cloud-init finish
	Eventually(func() string {
		output := runCMDNoWait(check_cloudFinal)
		return output
	}, 20*time.Minute, 30*time.Second).Should(ContainSubstring("Active: active (exited)"))

	GinkgoWriter.Println("cloud-init has finished")

	// ssh into the VM and check installed app
	output = runCMDNoWait(check_gitVersion)
	Expect(output).Should(ContainSubstring("git version"))

	// ssh into the VM and check file created
	output = runCMDNoWait(check_file_exist)
	Expect(output).Should(ContainSubstring("Hello from cloud-init"))
}

func init_vm_no_param_test(imagePath string, opts ...string) {
	initCMD := []string{"init", imagePath}
	startCMD := []string{"start"}
	checkUser := []string{"ssh", "whoami"}

	if len(opts) > 0 {
		provider := []string{"--provider", opts[0]}
		initCMD = append(initCMD, provider...)
		startCMD = append(startCMD, provider...)
		checkUser = append(provider, checkUser...)
	}

	runCMDsuccess(initCMD)

	// check the list command returns one item
	vmNumberCheck(1, opts...)

	// start the VM
	output := runCMDsuccess(startCMD)
	Expect(output).Should(ContainSubstring("started successfully"))

	// ssh into the VM and prints user
	output = runCMDNoWait(checkUser)
	expectedUser := "core"
	if len(opts) > 0 && opts[0] == "wsl" {
		expectedUser = "user"
	}
	Expect(strings.TrimSpace(output)).Should(Equal(expectedUser))
}
