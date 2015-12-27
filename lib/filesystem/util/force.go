package util

import (
	"os"
)

func forceRename(oldpath, newpath string) error {
	err := os.Rename(oldpath, newpath)
	if err == nil {
		return nil
	}
	if os.IsPermission(err) {
		// Blindly attempt to remove immutable attribute.
		MakeMutable(oldpath)
	}
	return os.Rename(oldpath, newpath)
}
