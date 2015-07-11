package scanner

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"io"
	"syscall"
	"time"
)

type FileSystemHistory struct {
	fileSystem         *FileSystem
	scanCount          uint64
	generationCount    uint64
	timeOfLastScan     time.Time
	durationOfLastScan time.Duration
	timeOfLastChange   time.Time
}

func (fsh *FileSystemHistory) Update(newFS *FileSystem) {
	fsh.update(newFS)
}

func (fsh *FileSystemHistory) FileSystem() *FileSystem {
	return fsh.fileSystem
}

func (fsh *FileSystemHistory) GenerationCount() uint64 {
	return fsh.generationCount
}

func (fsh FileSystemHistory) String() string {
	return fmt.Sprintf("GenerationCount=%d\n", fsh.generationCount)
}

func (fsh *FileSystemHistory) WriteHtml(writer io.Writer) {
	fsh.writeHtml(writer)
}

type RegularInodeTable map[uint64]*RegularInode
type InodeTable map[uint64]*Inode
type InodeList map[uint64]bool

type FileSystem struct {
	ctx                *fsrateio.FsRateContext
	RegularInodeTable  RegularInodeTable
	InodeTable         InodeTable // This excludes directories.
	DirectoryInodeList InodeList
	TotalDataBytes     uint64
	HashCount          uint64
	ObjectCache        [][]byte
	Dev                uint64
	Directory
}

func ScanFileSystem(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext) (*FileSystem, error) {
	return scanFileSystem(rootDirectoryName, cacheDirectoryName, ctx, nil)
}

func (fs *FileSystem) RebuildPointers() {
	fs.rebuildPointers()
}

func (fs *FileSystem) String() string {
	return fmt.Sprintf("Tree: %d inodes, total file size: %s, number of hashes: %d\nObjectCache: %d objects\n",
		len(fs.RegularInodeTable)+len(fs.InodeTable)+len(fs.DirectoryInodeList),
		fsrateio.FormatBytes(fs.TotalDataBytes),
		fs.HashCount,
		len(fs.ObjectCache))
}

func (fs *FileSystem) WriteHtml(writer io.Writer) {
	fs.writeHtml(writer)
}

func (fs *FileSystem) DebugWrite(w io.Writer, prefix string) error {
	return fs.debugWrite(w, prefix)
}

type Directory struct {
	Name            string
	RegularFileList []*RegularFile
	FileList        []*File
	DirectoryList   []*Directory
	Mode            uint32
	Uid             uint32
	Gid             uint32
}

func (directory *Directory) String() string {
	return directory.Name
}

func (directory *Directory) DebugWrite(w io.Writer, prefix string) error {
	return directory.debugWrite(w, prefix)
}

type RegularInode struct {
	Mode  uint32
	Uid   uint32
	Gid   uint32
	Size  uint64
	Mtime syscall.Timespec
	Hash  [64]byte
}

type RegularFile struct {
	Name        string
	InodeNumber uint64
	inode       *RegularInode
}

func (file *RegularFile) String() string {
	return file.Name
}

func (file *RegularFile) DebugWrite(w io.Writer, prefix string) error {
	return file.debugWrite(w, prefix)
}

type Inode struct {
	Mode    uint32
	Uid     uint32
	Gid     uint32
	Rdev    uint64
	Size    uint64
	Mtime   syscall.Timespec
	Symlink string
}

type File struct {
	Name        string
	InodeNumber uint64
	inode       *Inode
}

func (file *File) String() string {
	return file.Name
}

func (file *File) DebugWrite(w io.Writer, prefix string) error {
	return file.debugWrite(w, prefix)
}

func Compare(left *FileSystem, right *FileSystem, logWriter io.Writer) bool {
	return compare(left, right, logWriter)
}

func StartScannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext) chan *FileSystem {
	return startScannerDaemon(rootDirectoryName, cacheDirectoryName, ctx)
}
