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

type FileSystem struct {
	ctx            *fsrateio.FsRateContext
	InodeTable     map[uint64]*Inode
	TotalDataBytes uint64
	ObjectCache    [][]byte
	Directory
}

func ScanFileSystem(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext) (*FileSystem, error) {
	return scanFileSystem(rootDirectoryName, cacheDirectoryName, ctx)
}

func (fs *FileSystem) String() string {
	return fmt.Sprintf("Tree: %d inodes\nObjectCache: %d objects\n",
		len(fs.InodeTable), len(fs.ObjectCache))
}

func (fs *FileSystem) DebugWrite(w io.Writer, prefix string) error {
	return fs.debugWrite(w, prefix)
}

type Inode struct {
	Stat    syscall.Stat_t
	Symlink string
	Hash    []byte
}

func (inode *Inode) Length() uint64 {
	return uint64(inode.Stat.Size)
}

type Directory struct {
	Name          string
	Inode         *Inode
	FileList      []*File
	DirectoryList []*Directory
}

func (directory *Directory) String() string {
	return directory.Name
}

func (directory *Directory) DebugWrite(w io.Writer, prefix string) error {
	return directory.debugWrite(w, prefix)
}

type File struct {
	Name  string
	Inode *Inode
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
