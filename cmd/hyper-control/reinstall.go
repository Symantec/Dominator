package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Symantec/Dominator/lib/log"
)

func reinstallSubcommand(args []string, logger log.DebugLogger) {
	err := reinstall(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reinstalling: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func reinstall(logger log.DebugLogger) error {
	kexec, err := exec.LookPath("kexec")
	if err != nil {
		return err
	}
	cmd := exec.Command("hostname", "-f")
	var hostname string
	if stdout, err := cmd.Output(); err != nil {
		return err
	} else {
		hostname = strings.TrimSpace(string(stdout))
	}
	rootDir, err := ioutil.TempDir("", "kexec")
	if err != nil {
		return err
	}
	defer os.RemoveAll(rootDir)
	_, initrdFile, err := makeInstallerDirectory(hostname, rootDir, logger)
	if err != nil {
		return err
	}
	logger.Println("running kexec in 5 seconds...")
	time.Sleep(time.Second * 5)
	var command string
	var args []string
	if os.Geteuid() == 0 {
		command = kexec
	} else {
		command = "sudo"
		args = []string{kexec}
	}
	args = append(args, "-l", filepath.Join(rootDir, "vmlinuz"),
		"--append=console=tty0 console=ttyS0,115200n8",
		"--console-serial", "--serial-baud=115200",
		"--initrd="+initrdFile, "-f")
	cmd = exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
