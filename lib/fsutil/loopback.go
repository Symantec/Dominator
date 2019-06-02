package fsutil

import (
	"fmt"
	"os/exec"
	"strings"
)

func loopbackDelete(loopDevice string) error {
	return exec.Command("losetup", "-d", loopDevice).Run()
}

func loopbackSetup(filename string) (string, error) {
	cmd := exec.Command("losetup", "-fP", "--show", filename)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, output)
	}
	return strings.TrimSpace(string(output)), nil
}
