package util

import (
	"os/exec"
)

func makeMutable(pathname string) error {
	// Blindly attempt to remove immutable attribute.
	cmd := exec.Command("chattr", "-ai", pathname)
	return cmd.Run()
}
