package fsutil

import (
	"os"
)

func forceRename(oldpath, newpath string) error {
	err := os.Rename(oldpath, newpath)
	if err == nil {
		return nil
	}
	if os.IsPermission(err) {
		// Blindly attempt to remove immutable attributes.
		MakeMutable(oldpath, newpath)
	}
	return os.Rename(oldpath, newpath)
}

func forceRemove(name string) error {
	err := os.Remove(name)
	if err == nil {
		return nil
	}
	if os.IsPermission(err) {
		// Blindly attempt to remove immutable attributes.
		MakeMutable(name)
	}
	return os.Remove(name)
}

func forceRemoveAll(path string) error {
	err := os.RemoveAll(path)
	if err == nil {
		return nil
	}
	if os.IsPermission(err) {
		// Blindly attempt to remove immutable attributes.
		MakeMutable(path)
	}
	return os.RemoveAll(path)
}
