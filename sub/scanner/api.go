package scanner

import (
	"github.com/Symantec/Dominator/sub/fsrateio"
	"io"
	"syscall"
)

type FileSystem struct {
	ctx        *fsrateio.FsRateContext
	InodeTable map[uint64]*Inode
	Directory
}

func ScanFileSystem(rootDirectoryName string,
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
	return &fileSystem, nil
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
