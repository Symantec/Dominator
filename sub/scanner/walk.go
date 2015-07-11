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

func (fileSystem *FileSystem) getRegularInode(stat *syscall.Stat_t) (
	*RegularInode, bool) {
	inode := fileSystem.RegularInodeTable[stat.Ino]
	new := false
	if inode == nil {
		var _inode RegularInode
		inode = &_inode
		_inode.Mode = stat.Mode
		_inode.Uid = stat.Uid
		_inode.Gid = stat.Gid
		_inode.Size = uint64(stat.Size)
		_inode.Mtime = stat.Mtim
		fileSystem.RegularInodeTable[stat.Ino] = inode
		new = true
	}
	return inode, new
}

func (fileSystem *FileSystem) getSymlinkInode(stat *syscall.Stat_t) (
	*SymlinkInode, bool) {
	inode := fileSystem.SymlinkInodeTable[stat.Ino]
	new := false
	if inode == nil {
		var _inode SymlinkInode
		inode = &_inode
		_inode.Uid = stat.Uid
		_inode.Gid = stat.Gid
		fileSystem.SymlinkInodeTable[stat.Ino] = inode
		new = true
	}
	return inode, new
}

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
		_inode.Mtime = stat.Mtim
		fileSystem.InodeTable[stat.Ino] = inode
		new = true
	}
	return inode, new
}

func scanFileSystem(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext, oldFS *FileSystem) (*FileSystem, error) {
	var fileSystem FileSystem
	fileSystem.ctx = ctx
	fileSystem.Name = rootDirectoryName
	var stat syscall.Stat_t
	err := syscall.Lstat(rootDirectoryName, &stat)
	if err != nil {
		return nil, err
	}
	fileSystem.RegularInodeTable = make(RegularInodeTable)
	fileSystem.SymlinkInodeTable = make(SymlinkInodeTable)
	fileSystem.InodeTable = make(InodeTable)
	fileSystem.DirectoryInodeList = make(InodeList)
	fileSystem.DirectoryInodeList[stat.Ino] = true
	fileSystem.Dev = stat.Dev
	fileSystem.Mode = stat.Mode
	fileSystem.Uid = stat.Uid
	fileSystem.Gid = stat.Gid
	var tmpInode RegularInode
	if sha512.New().Size() != len(tmpInode.Hash) {
		return nil, errors.New("Incompatible hash size")
	}
	err = fileSystem.scan(&fileSystem, oldFS, "")
	oldFS = nil
	fileSystem.DirectoryInodeList = nil
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
	for _, inode := range fs.RegularInodeTable {
		totalBytes += uint64(inode.Size)
	}
	return totalBytes
}

func (directory *Directory) scan(fileSystem, oldFS *FileSystem,
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
	directory.FileList = make([]*File, 0, len(names))
	directory.DirectoryList = make([]*Directory, 0, len(names))
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
		if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
			err = directory.addDirectory(fileSystem, oldFS, name, myPathName,
				&stat)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFREG {
			err = directory.addRegularFile(fileSystem, oldFS, name, myPathName,
				&stat)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFLNK {
			err = directory.addSymlink(fileSystem, oldFS, name, myPathName,
				&stat)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFSOCK {
			continue
		} else {
			err = directory.addFile(fileSystem, oldFS, name, myPathName, &stat)
		}
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return nil
		}
	}
	// Save file and directory lists which are exactly the right length.
	regularFileList := make([]*RegularFile, len(directory.RegularFileList))
	copy(regularFileList, directory.RegularFileList)
	directory.RegularFileList = regularFileList
	symlinkList := make([]*Symlink, len(directory.SymlinkList))
	copy(symlinkList, directory.SymlinkList)
	directory.SymlinkList = symlinkList
	fileList := make([]*File, len(directory.FileList))
	copy(fileList, directory.FileList)
	directory.FileList = fileList
	directoryList := make([]*Directory, len(directory.DirectoryList))
	copy(directoryList, directory.DirectoryList)
	directory.DirectoryList = directoryList
	return nil
}

func (directory *Directory) addDirectory(fileSystem, oldFS *FileSystem,
	name string, directoryPathName string, stat *syscall.Stat_t) error {
	myPathName := path.Join(directoryPathName, name)
	if fileSystem.DirectoryInodeList[stat.Ino] {
		return errors.New("Hardlinked directory: " + myPathName)
	}
	fileSystem.DirectoryInodeList[stat.Ino] = true
	var dir Directory
	dir.Name = name
	dir.Mode = stat.Mode
	dir.Uid = stat.Uid
	dir.Gid = stat.Gid
	err := dir.scan(fileSystem, oldFS, directoryPathName)
	if err != nil {
		return err
	}
	directory.DirectoryList = append(directory.DirectoryList, &dir)
	return nil
}

func (directory *Directory) addRegularFile(fileSystem, oldFS *FileSystem,
	name string, directoryPathName string, stat *syscall.Stat_t) error {
	inode, isNewInode := fileSystem.getRegularInode(stat)
	var file RegularFile
	file.Name = name
	file.InodeNumber = stat.Ino
	file.inode = inode
	if isNewInode {
		err := file.scan(fileSystem, directoryPathName)
		if err != nil {
			return err
		}
		if oldFS != nil && oldFS.RegularInodeTable != nil {
			if oldInode, found := oldFS.RegularInodeTable[stat.Ino]; found {
				if compareRegularInodes(inode, oldInode, nil) {
					inode = oldInode
					file.inode = inode
					fileSystem.RegularInodeTable[stat.Ino] = inode
				}
			}
		}
	}
	directory.RegularFileList = append(directory.RegularFileList, &file)
	return nil
}

func (directory *Directory) addSymlink(fileSystem, oldFS *FileSystem,
	name string, directoryPathName string, stat *syscall.Stat_t) error {
	inode, isNewInode := fileSystem.getSymlinkInode(stat)
	var symlink Symlink
	symlink.Name = name
	symlink.InodeNumber = stat.Ino
	symlink.inode = inode
	if isNewInode {
		err := symlink.scan(fileSystem, directoryPathName)
		if err != nil {
			return err
		}
		if oldFS != nil && oldFS.SymlinkInodeTable != nil {
			if oldInode, found := oldFS.SymlinkInodeTable[stat.Ino]; found {
				if compareSymlinkInodes(inode, oldInode, nil) {
					inode = oldInode
					symlink.inode = inode
					fileSystem.SymlinkInodeTable[stat.Ino] = inode
				}
			}
		}
	}
	directory.SymlinkList = append(directory.SymlinkList, &symlink)
	return nil
}

func (directory *Directory) addFile(fileSystem, oldFS *FileSystem, name string,
	directoryPathName string, stat *syscall.Stat_t) error {
	inode, isNewInode := fileSystem.getInode(stat)
	var file File
	file.Name = name
	file.InodeNumber = stat.Ino
	file.inode = inode
	if isNewInode {
		err := file.scan(fileSystem, directoryPathName)
		if err != nil {
			return err
		}
		if oldFS != nil && oldFS.InodeTable != nil {
			if oldInode, found := oldFS.InodeTable[stat.Ino]; found {
				if compareInodes(inode, oldInode, nil) {
					inode = oldInode
					file.inode = inode
					fileSystem.InodeTable[stat.Ino] = inode
				}
			}
		}
	}
	directory.FileList = append(directory.FileList, &file)
	return nil
}

func (file *RegularFile) scan(fileSystem *FileSystem, parentName string) error {
	myPathName := path.Join(parentName, file.Name)
	f, err := os.Open(myPathName)
	if err != nil {
		return err
	}
	reader := fsrateio.NewReader(f, fileSystem.ctx)
	hash := sha512.New()
	io.Copy(hash, reader)
	f.Close()
	copy(file.inode.Hash[:], hash.Sum(nil))
	fileSystem.HashCount++
	return nil
}

func (symlink *Symlink) scan(fileSystem *FileSystem, parentName string) error {
	myPathName := path.Join(parentName, symlink.Name)
	target, err := os.Readlink(myPathName)
	if err != nil {
		return err
	}
	symlink.inode.Symlink = target
	return nil
}

func (file *File) scan(fileSystem *FileSystem, parentName string) error {
	return nil
}
