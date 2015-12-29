package fsutil

import (
	"io"
)

// CopyToFile will create a new file, writre length bytes from reader to the
// file and then atimically renames the file to destFilename. If there are any
// errors, then destFilename is unchanged.
func CopyToFile(destFilename string, reader io.Reader, length int64) error {
	return copyToFile(destFilename, reader, length)
}

// ForceLink creates newname as a hard link to the oldname file. It first
// attempts to link using os.Link and if that fails, it blindly calls
// MakeMutable and then retries.
func ForceLink(oldname, newname string) error {
	return forceLink(oldname, newname)
}

// ForceRemove removes the named file or directory. It first attempts to remove
// using os.Remove and that fails, it blindly calls MakeMutable and then
// retries.
func ForceRemove(name string) error {
	return forceRemove(name)
}

// ForceRemoveAll removes path and any children it contains. It first attempts
// to remove using os.RemoveAll and that fails, it blindly calls MakeMutable and
// then retries.
func ForceRemoveAll(path string) error {
	return forceRemoveAll(path)
}

// ForceRename renames (moves) a file. It first attempts to rename using
// os.Rename and if that fails, it blindly calls MakeMutable and then retries.
func ForceRename(oldpath, newpath string) error {
	return forceRename(oldpath, newpath)
}

// MakeMutable attempts to remove the "immutable" and "append-only" ext2
// file-system attributes for one or more files. It is equivalent to calling the
// command-line programme "chattr -ai pathname...".
func MakeMutable(pathname ...string) error {
	return makeMutable(pathname...)
}
