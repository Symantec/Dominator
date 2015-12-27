package fsutil

import (
	"os/exec"
)

func makeMutable(pathname ...string) error {
	args := make([]string, 0, len(pathname)+1)
	args = append(args, "-ai")
	args = append(args, pathname...)
	cmd := exec.Command("chattr", args...)
	return cmd.Run()
}
