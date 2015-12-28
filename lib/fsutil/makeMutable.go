package fsutil

import (
	"os/exec"
)

func makeMutable(pathname ...string) error {
	args := make([]string, 0, len(pathname)+2)
	args = append(args, "-R")
	args = append(args, "-ai")
	args = append(args, pathname...)
	cmd := exec.Command("chattr", args...)
	return cmd.Run()
}
