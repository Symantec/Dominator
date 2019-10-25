package main

import (
	"os"
	"os/exec"

	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func runShellOnConsole(logger log.DebugLogger) {
	for {
		logger.Println("starting shell on console")
		cmd := exec.Command("/bin/busybox", "sh", "-i")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			logger.Printf("error running shell: %s\n", err)
		}
	}
}
