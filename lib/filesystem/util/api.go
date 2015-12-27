package util

// MakeMutable attempts to remove the "immutable" and "append-only" ext2
// file-system attributes for a file. It is equivalent to calling the
// command-line programme "chattr -ai pathname".
func MakeMutable(pathname string) error {
	return makeMutable(pathname)
}

// ForceRename renames (moves) a file. It first attempts to rename using
// os.Rename and if that fails, it blindly calls MakeMutable and then retries.
func ForceRename(oldpath, newpath string) error {
	return forceRename(oldpath, newpath)
}
