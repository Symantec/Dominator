package fsutil

import (
	"errors"
	"hash"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/log"
)

const (
	DirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
	PrivateFilePerms = syscall.S_IRUSR | syscall.S_IWUSR
	PublicFilePerms  = PrivateFilePerms | syscall.S_IRGRP | syscall.S_IROTH
)

var (
	ErrorChecksumMismatch = errors.New("checksum mismatch")
)

// CompareFiles will read and compare the content of two files and return true
// if they are the same else false.
func CompareFiles(leftFilename, rightFilename string) (bool, error) {
	return compareFiles(leftFilename, rightFilename)
}

func CopyFile(destFilename, sourceFilename string, mode os.FileMode) error {
	return copyFile(destFilename, sourceFilename, mode)
}

// CopyToFile will create a new file, write length bytes from reader to the
// file and then atomically renames the file to destFilename. If length is zero
// all remaining bytes from reader are written. If there are any errors, then
// destFilename is unchanged.
func CopyToFile(destFilename string, perm os.FileMode, reader io.Reader,
	length uint64) error {
	return copyToFile(destFilename, perm, reader, length)
}

// CopyTree will copy a directory tree.
func CopyTree(destDir, sourceDir string) error {
	return copyTree(destDir, sourceDir, copyFile)
}

// CopyTreeWithCopyFunc is similar to CopyTree except it uses a specified copy
// function for copying regular files.
func CopyTreeWithCopyFunc(destDir, sourceDir string,
	copyFunc func(destFilename, sourceFilename string,
		mode os.FileMode) error) error {
	return copyTree(destDir, sourceDir, copyFunc)
}

// ForceLink creates newname as a hard link to the oldname file. It first
// attempts to link using os.Link. If the first attempt fails due to
// a permission error, it blindly calls MakeMutable and then retries. If the
// first attempt fails due to newname existing, it blindly removes it and then
// retries.
func ForceLink(oldname, newname string) error {
	return forceLink(oldname, newname)
}

// Fallocate will allocate blocks for the file named filename, up to size
// specified in bytes.
func Fallocate(filename string, size uint64) error {
	return fallocate(filename, size)
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
// os.Rename and if that fails due to a permission error, it blindly calls
// MakeMutable and then retries. If it fails because newpath is a directory, it
// calls ForceRemoveAll(newpath) and tries again.
func ForceRename(oldpath, newpath string) error {
	return forceRename(oldpath, newpath)
}

// LoadLines will open a file and read lines from it. Comment lines (i.e. lines
// beginning with '#') are skipped.
func LoadLines(filename string) ([]string, error) {
	return loadLines(filename)
}

// MakeMutable attempts to remove the "immutable" and "append-only" ext2
// file-system attributes for one or more files. It is equivalent to calling the
// command-line programme "chattr -ai pathname...".
func MakeMutable(pathname ...string) error {
	return makeMutable(pathname...)
}

// ReadDirnames will open the directory named dirname and will read the entries
// in that directory. If ignoreMissing is true, no error is returned if the
// directory does not exist.
func ReadDirnames(dirname string, ignoreMissing bool) ([]string, error) {
	return readDirnames(dirname, ignoreMissing)
}

// WaitFile waits for the file given by pathname to become available to read and
// yields a io.ReadCloser when available, or an error if the timeout is
// exceeded or an error (other than file not existing) is encountered. A
// negative timeout indicates to wait forever. The io.ReadCloser must be closed
// after use.
func WaitFile(pathname string, timeout time.Duration) (io.ReadCloser, error) {
	return waitFile(pathname, timeout)
}

// WatchFile watches the file given by pathname and yields a new io.ReadCloser
// when a new inode is found and it is a regular file. The io.ReadCloser must
// be closed after use.
// Any errors are logged to the logger if it is not nil.
func WatchFile(pathname string, logger log.Logger) <-chan io.ReadCloser {
	return watchFile(pathname, logger)
}

// WatchFileStop stops all file watching and cleans up resources that would
// otherwise persist across syscall.Exec.
func WatchFileStop() {
	watchFileStop()
}

type ChecksumReader struct {
	checksummer hash.Hash
	reader      io.Reader
}

func NewChecksumReader(reader io.Reader) *ChecksumReader {
	return newChecksumReader(reader)
}

func (r *ChecksumReader) GetChecksum() []byte {
	return r.getChecksum()
}

func (r *ChecksumReader) Read(p []byte) (int, error) {
	return r.read(p)
}

func (r *ChecksumReader) ReadByte() (byte, error) {
	return r.readByte()
}

func (r *ChecksumReader) VerifyChecksum() error {
	return r.verifyChecksum()
}

type ChecksumWriter struct {
	checksummer hash.Hash
	writer      io.Writer
}

func NewChecksumWriter(writer io.Writer) *ChecksumWriter {
	return newChecksumWriter(writer)
}

func (w *ChecksumWriter) Write(p []byte) (int, error) {
	return w.write(p)
}

func (w *ChecksumWriter) WriteChecksum() error {
	return w.writeChecksum()
}

type RenamingWriter struct {
	*os.File
	filename string
	abort    bool
}

func CreateRenamingWriter(filename string, perm os.FileMode) (
	*RenamingWriter, error) {
	return createRenamingWriter(filename, perm)
}

func (w *RenamingWriter) Close() error {
	if err := recover(); err != nil {
		w.abort = true
		w.close()
		panic(err)
	}
	return w.close()
}

func (w *RenamingWriter) Write(p []byte) (int, error) {
	return w.write(p)
}
