package scanner

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"io"
	"syscall"
)

type FileSystemHistory struct {
	fileSystem      *FileSystem
	generationCount uint64
}

func (fsh *FileSystemHistory) Update(newFS *FileSystem) {
	if fsh.fileSystem == nil {
		fsh.fileSystem = newFS
		fsh.generationCount = 1
	} else {
		if !Compare(fsh.fileSystem, newFS, nil) {
			fsh.generationCount++
			fsh.fileSystem = newFS
		}
	}
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

type FileSystem struct {
	ctx         *fsrateio.FsRateContext
	InodeTable  map[uint64]*Inode
	ObjectCache [][]byte
	Directory
}

func ScanFileSystem(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext) (*FileSystem, error) {
	var fileSystem FileSystem
	fileSystem.ctx = ctx
	fileSystem.name = rootDirectoryName
	var stat syscall.Stat_t
	err := syscall.Lstat(rootDirectoryName, &stat)
	if err != nil {
		return nil, err
	}
	fileSystem.InodeTable = make(map[uint64]*Inode)
	fileSystem.inode, _ = fileSystem.getInode(&stat)
	err = fileSystem.scan(&fileSystem, "")
	if err != nil {
		return nil, err
	}
	if cacheDirectoryName != "" {
		fileSystem.ObjectCache = make([][]byte, 0, 65536)
		fileSystem.ObjectCache, err = scanObjectCache(cacheDirectoryName, "",
			fileSystem.ObjectCache)
		if err != nil {
			return nil, err
		}
	}
	return &fileSystem, nil
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
