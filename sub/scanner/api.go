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
	ctx         *fsrateio.FsRateContext
	InodeTable  map[uint64]*Inode
	ObjectCache [][]byte
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

type Inode struct {
	stat    syscall.Stat_t
	symlink string
	hash    []byte
}

func (inode *Inode) Length() uint64 {
	return uint64(inode.stat.Size)
}

type Directory struct {
	name          string
	inode         *Inode
	FileList      []*File
	DirectoryList []*Directory
}

func (directory *Directory) Name() string {
	return directory.name
}

func (directory *Directory) String() string {
	return directory.name
}

type File struct {
	name  string
	inode *Inode
}

func (file *File) Name() string {
	return file.name
}

func (file *File) String() string {
	return file.name
}

func Compare(left *FileSystem, right *FileSystem, logWriter io.Writer) bool {
	return compare(left, right, logWriter)
}

func StartScannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext) chan *FileSystem {
	return startScannerDaemon(rootDirectoryName, cacheDirectoryName, ctx)
}
