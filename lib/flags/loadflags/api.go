package loadflags

import (
	"os"
	"path/filepath"
)

func LoadForCli(progName string) error {
	return loadFlags(
		filepath.Join(os.Getenv("HOME"), ".config", progName, "flags.default"))
}

func LoadForDaemon(progName string) error {
	return loadFlags(filepath.Join("/etc", progName, "flags.default"))
}
