package scanner

import (
	"crypto/sha512"
	"errors"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"io"
	"os"
	"path"
	"sort"
	"syscall"
)

func (fileSystem *FileSystem) getInode(stat *syscall.Stat_t) (*Inode, bool) {
	inode := fileSystem.InodeTable[stat.Ino]
	new := false
	if inode == nil {
		var _inode Inode
		inode = &_inode
		_inode.Mode = stat.Mode
		_inode.Uid = stat.Uid
		_inode.Gid = stat.Gid
		_inode.Rdev = stat.Rdev
		_inode.Size = uint64(stat.Size)
		_inode.Mtime = stat.Mtim
		fileSystem.InodeTable[stat.Ino] = inode
		new = true
	}
	return inode, new
}

func scanFileSystem(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext) (*FileSystem, error) {
	var fileSystem FileSystem
	fileSystem.ctx = ctx
	fileSystem.Name = rootDirectoryName
	var stat syscall.Stat_t
	err := syscall.Lstat(rootDirectoryName, &stat)
	if err != nil {
		return nil, err
	}
	fileSystem.InodeTable = make(map[uint64]*Inode)
	fileSystem.Dev = stat.Dev
	fileSystem.InodeNumber = stat.Ino
	fileSystem.inode, _ = fileSystem.getInode(&stat)
	err = fileSystem.scan(&fileSystem, "")
	if err != nil {
		return nil, err
	}
	if cacheDirectoryName != "" {
		fileSystem.ObjectCache = make([][]byte, 0, 16)
		fileSystem.ObjectCache, err = scanObjectCache(cacheDirectoryName, "",
			fileSystem.ObjectCache)
		if err != nil {
			return nil, err
		}
	}
	fileSystem.TotalDataBytes = fileSystem.computeTotalDataBytes()
	return &fileSystem, nil
}

func (fs *FileSystem) computeTotalDataBytes() uint64 {
	var totalBytes uint64 = 0
	for _, inode := range fs.InodeTable {
		totalBytes += uint64(inode.Size)
	}
	return totalBytes
}

func (directory *Directory) scan(fileSystem *FileSystem,
	parentName string) error {
	myPathName := path.Join(parentName, directory.Name)
	file, err := os.Open(myPathName)
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	if err != nil {
		return err
	}
	file.Close()
	sort.Strings(names)
	// Create file and directory lists which are guaranteed to be long enough.
	fileList := make([]*File, 0, len(names))
	directoryList := make([]*Directory, 0, len(names))
	for _, name := range names {
		filename := path.Join(myPathName, name)
		var stat syscall.Stat_t
		err := syscall.Lstat(filename, &stat)
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return err
		}
		if stat.Dev != fileSystem.Dev {
			continue
		}
		inode, isNewInode := fileSystem.getInode(&stat)
		if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
			if !isNewInode {
				return errors.New("Hardlinked directory: " + filename)
			}
			var dir Directory
			dir.Name = name
			dir.InodeNumber = stat.Ino
			dir.inode = inode
			err := dir.scan(fileSystem, myPathName)
			if err != nil {
				return err
			}
			directoryList = append(directoryList, &dir)
		} else {
			var file File
			file.Name = name
			file.InodeNumber = stat.Ino
			file.inode = inode
			if isNewInode {
				err := file.scan(fileSystem, myPathName)
				if err != nil {
					if err == syscall.ENOENT {
						continue
					}
					return err
				}
			}
			fileList = append(fileList, &file)
		}
	}
	// Save file and directory lists which are exactly the right length.
	directory.FileList = make([]*File, len(fileList))
	copy(directory.FileList, fileList)
	directory.DirectoryList = make([]*Directory, len(directoryList))
	copy(directory.DirectoryList, directoryList)
	return nil
}

func (file *File) scan(fileSystem *FileSystem, parentName string) error {
	myPathName := path.Join(parentName, file.Name)
	if file.inode.Mode&syscall.S_IFMT == syscall.S_IFREG {
		f, err := os.Open(myPathName)
		if err != nil {
			return err
		}
		reader := fsrateio.NewReader(f, fileSystem.ctx)
		hash := sha512.New()
		io.Copy(hash, reader)
		f.Close()
		file.inode.Hash = hash.Sum(nil)
		fileSystem.HashCount++
	} else if file.inode.Mode&syscall.S_IFMT == syscall.S_IFLNK {
		symlink, err := os.Readlink(myPathName)
		if err != nil {
			return err
		}
		file.inode.Symlink = symlink
	}
	return nil
}
