package scanner

import (
	"crypto/sha512"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"io"
	"os"
	"path"
	"syscall"
)

func (fileSystem *FileSystem) getInode(stat *syscall.Stat_t) *Inode {
	inode := fileSystem.InodeTable[stat.Ino]
	if inode == nil {
		var _inode Inode
		inode = &_inode
		_inode.stat = *stat
		fileSystem.InodeTable[stat.Ino] = inode
	}
	return inode
}

func (directory *Directory) scan(fileSystem *FileSystem,
	parentName string) error {
	myPathName := path.Join(parentName, directory.name)
	file, err := os.Open(myPathName)
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	if err != nil {
		return err
	}
	file.Close()
	for _, name := range names {
		filename := path.Join(myPathName, name)
		var stat syscall.Stat_t
		err := syscall.Lstat(filename, &stat)
		if err != nil {
			return err
		}
		inode := fileSystem.getInode(&stat)
		if stat.Dev == directory.inode.stat.Dev {
			if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
				var dir Directory
				dir.name = name
				dir.inode = inode
				err := dir.scan(fileSystem, myPathName)
				if err != nil {
					return err
				}
				directory.DirectoryList = append(directory.DirectoryList, &dir)
			} else {
				var file File
				file.name = name
				file.inode = inode
				err := file.scan(fileSystem, myPathName)
				if err != nil {
					return err
				}
				directory.FileList = append(directory.FileList, &file)
			}
		}
	}
	return nil
}

func (file *File) scan(fileSystem *FileSystem, parentName string) error {
	myPathName := path.Join(parentName, file.name)
	if file.inode.stat.Mode&syscall.S_IFMT != syscall.S_IFREG {
		return nil
	}
	if len(file.inode.hash) > 0 {
		return nil
	}
	f, err := os.Open(myPathName)
	if err != nil {
		return err
	}
	reader := fsrateio.NewReader(f, fileSystem.ctx)
	hash := sha512.New()
	io.Copy(hash, reader)
	f.Close()
	file.inode.hash = hash.Sum(nil)
	return nil
}
