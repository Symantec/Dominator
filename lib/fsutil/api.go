package fsutil

import (
	"errors"
	"github.com/Symantec/Dominator/lib/log"
	"hash"
	"io"
	"os"
	"time"
)

var (
	ErrorChecksumMismatch = errors.New("checksum mismatch")
)

// CopyToFile will create a new file, write length bytes from reader to the
// file and then atomically renames the file to destFilename. If there are any
// errors, then destFilename is unchanged.
func CopyToFile(destFilename string, perm os.FileMode, reader io.Reader,
	length uint64) error {
	return copyToFile(destFilename, perm, reader, length)
}

// ForceLink creates newname as a hard link to the oldname file. It first
// attempts to link using os.Link. If the first attempt fails due to
// a permission error, it blindly calls MakeMutable and then retries. If the
// first attempt fails due to newname existing, it blindly removes it and then
// retries.
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
}

func CreateRenamingWriter(filename string, perm os.FileMode) (
	*RenamingWriter, error) {
	return createRenamingWriter(filename, perm)
}

func (w *RenamingWriter) Close() error {
	return w.close()
}
