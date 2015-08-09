package scanner

import (
	"crypto/sha512"
	"errors"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"io"
	"os"
	"path"
	"sort"
	"syscall"
)

func (fileSystem *FileSystem) getRegularInode(stat *syscall.Stat_t) (
	*filesystem.RegularInode, bool) {
	inode := fileSystem.RegularInodeTable[stat.Ino]
	new := false
	if inode == nil {
		var _inode filesystem.RegularInode
		inode = &_inode
		_inode.Mode = stat.Mode
		_inode.Uid = stat.Uid
		_inode.Gid = stat.Gid
		_inode.MtimeSeconds = stat.Mtim.Sec
		_inode.MtimeNanoSeconds = int32(stat.Mtim.Nsec)
		_inode.Size = uint64(stat.Size)
		fileSystem.RegularInodeTable[stat.Ino] = inode
		new = true
	}
	return inode, new
}

func (fileSystem *FileSystem) getSymlinkInode(stat *syscall.Stat_t) (
	*filesystem.SymlinkInode, bool) {
	inode := fileSystem.SymlinkInodeTable[stat.Ino]
	new := false
	if inode == nil {
		var _inode filesystem.SymlinkInode
		inode = &_inode
		_inode.Uid = stat.Uid
		_inode.Gid = stat.Gid
		fileSystem.SymlinkInodeTable[stat.Ino] = inode
		new = true
	}
	return inode, new
}

func (fileSystem *FileSystem) getInode(stat *syscall.Stat_t) (
	*filesystem.Inode, bool) {
	inode := fileSystem.InodeTable[stat.Ino]
	new := false
	if inode == nil {
		var _inode filesystem.Inode
		inode = &_inode
		_inode.Mode = stat.Mode
		_inode.Uid = stat.Uid
		_inode.Gid = stat.Gid
		_inode.MtimeSeconds = stat.Mtim.Sec
		_inode.MtimeNanoSeconds = int32(stat.Mtim.Nsec)
		_inode.Rdev = stat.Rdev
		fileSystem.InodeTable[stat.Ino] = inode
		new = true
	}
	return inode, new
}

func scanFileSystem(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration, oldFS *FileSystem) (*FileSystem, error) {
	var fileSystem FileSystem
	fileSystem.configuration = configuration
	fileSystem.rootDirectoryName = rootDirectoryName
	fileSystem.Name = "/"
	var stat syscall.Stat_t
	err := syscall.Lstat(rootDirectoryName, &stat)
	if err != nil {
		return nil, err
	}
	fileSystem.RegularInodeTable = make(filesystem.RegularInodeTable)
	fileSystem.SymlinkInodeTable = make(filesystem.SymlinkInodeTable)
	fileSystem.InodeTable = make(filesystem.InodeTable)
	fileSystem.directoryInodeList = make(directoryInodeList)
	fileSystem.directoryInodeList[stat.Ino] = true
	fileSystem.dev = stat.Dev
	fileSystem.Mode = stat.Mode
	fileSystem.Uid = stat.Uid
	fileSystem.Gid = stat.Gid
	var tmpInode filesystem.RegularInode
	if sha512.New().Size() != len(tmpInode.Hash) {
		return nil, errors.New("Incompatible hash size")
	}
	err = scanDirectory(&fileSystem.FileSystem.Directory, &fileSystem, oldFS,
		"")
	oldFS = nil
	fileSystem.DirectoryCount = uint64(len(fileSystem.directoryInodeList))
	fileSystem.directoryInodeList = nil
	if err != nil {
		return nil, err
	}
	if cacheDirectoryName != "" {
		fileSystem.ObjectCache, err = objectcache.ScanObjectCache(
			cacheDirectoryName)
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

func scanDirectory(directory *filesystem.Directory,
	fileSystem, oldFS *FileSystem, parentName string) error {
	myPathName := path.Join(parentName, directory.Name)
	file, err := os.Open(path.Join(fileSystem.rootDirectoryName, myPathName))
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		filename := path.Join(myPathName, name)
		skip := false
		for _, regex := range fileSystem.configuration.ExclusionList {
			if regex.MatchString(filename) {
				skip = true
				continue
			}
		}
		if skip {
			continue
		}
		var stat syscall.Stat_t
		err := syscall.Lstat(path.Join(fileSystem.rootDirectoryName, filename),
			&stat)
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return err
		}
		if stat.Dev != fileSystem.dev {
			continue
		}
		if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
			err = addDirectory(directory, fileSystem, oldFS, name, myPathName,
				&stat)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFREG {
			err = addRegularFile(directory, fileSystem, oldFS, name, myPathName,
				&stat)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFLNK {
			err = addSymlink(directory, fileSystem, oldFS, name, myPathName,
				&stat)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFSOCK {
			continue
		} else {
			err = addFile(directory, fileSystem, oldFS, name, myPathName, &stat)
		}
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return err
		}
	}
	// Save file and directory lists which are exactly the right length.
	regularFileList := make([]*filesystem.RegularFile,
		len(directory.RegularFileList))
	copy(regularFileList, directory.RegularFileList)
	directory.RegularFileList = regularFileList
	symlinkList := make([]*filesystem.Symlink, len(directory.SymlinkList))
	copy(symlinkList, directory.SymlinkList)
	directory.SymlinkList = symlinkList
	fileList := make([]*filesystem.File, len(directory.FileList))
	copy(fileList, directory.FileList)
	directory.FileList = fileList
	directoryList := make([]*filesystem.Directory, len(directory.DirectoryList))
	copy(directoryList, directory.DirectoryList)
	directory.DirectoryList = directoryList
	return nil
}

func addDirectory(directory *filesystem.Directory,
	fileSystem, oldFS *FileSystem,
	name string, directoryPathName string, stat *syscall.Stat_t) error {
	myPathName := path.Join(directoryPathName, name)
	if fileSystem.directoryInodeList[stat.Ino] {
		return errors.New("Hardlinked directory: " + myPathName)
	}
	fileSystem.directoryInodeList[stat.Ino] = true
	var dir filesystem.Directory
	dir.Name = name
	dir.Mode = stat.Mode
	dir.Uid = stat.Uid
	dir.Gid = stat.Gid
	err := scanDirectory(&dir, fileSystem, oldFS, directoryPathName)
	if err != nil {
		return err
	}
	directory.DirectoryList = append(directory.DirectoryList, &dir)
	return nil
}

func addRegularFile(directory *filesystem.Directory,
	fileSystem, oldFS *FileSystem,
	name string, directoryPathName string, stat *syscall.Stat_t) error {
	inode, isNewInode := fileSystem.getRegularInode(stat)
	var file filesystem.RegularFile
	file.Name = name
	file.InodeNumber = stat.Ino
	file.SetInode(inode)
	if isNewInode {
		err := scanRegularFile(&file, fileSystem, directoryPathName)
		if err != nil {
			return err
		}
		if oldFS != nil && oldFS.RegularInodeTable != nil {
			if oldInode, found := oldFS.RegularInodeTable[stat.Ino]; found {
				if filesystem.CompareRegularInodes(inode, oldInode, nil) {
					inode = oldInode
					file.SetInode(inode)
					fileSystem.RegularInodeTable[stat.Ino] = inode
				}
			}
		}
	}
	directory.RegularFileList = append(directory.RegularFileList, &file)
	return nil
}

func addSymlink(directory *filesystem.Directory, fileSystem, oldFS *FileSystem,
	name string, directoryPathName string, stat *syscall.Stat_t) error {
	inode, isNewInode := fileSystem.getSymlinkInode(stat)
	var symlink filesystem.Symlink
	symlink.Name = name
	symlink.InodeNumber = stat.Ino
	symlink.SetInode(inode)
	if isNewInode {
		err := scanSymlink(&symlink, fileSystem, directoryPathName)
		if err != nil {
			return err
		}
		if oldFS != nil && oldFS.SymlinkInodeTable != nil {
			if oldInode, found := oldFS.SymlinkInodeTable[stat.Ino]; found {
				if filesystem.CompareSymlinkInodes(inode, oldInode, nil) {
					inode = oldInode
					symlink.SetInode(inode)
					fileSystem.SymlinkInodeTable[stat.Ino] = inode
				}
			}
		}
	}
	directory.SymlinkList = append(directory.SymlinkList, &symlink)
	return nil
}

func addFile(directory *filesystem.Directory, fileSystem, oldFS *FileSystem,
	name string, directoryPathName string, stat *syscall.Stat_t) error {
	inode, isNewInode := fileSystem.getInode(stat)
	var file filesystem.File
	file.Name = name
	file.InodeNumber = stat.Ino
	file.SetInode(inode)
	if isNewInode {
		err := scanFile(&file, fileSystem, directoryPathName)
		if err != nil {
			return err
		}
		if oldFS != nil && oldFS.InodeTable != nil {
			if oldInode, found := oldFS.InodeTable[stat.Ino]; found {
				if filesystem.CompareInodes(inode, oldInode, nil) {
					inode = oldInode
					file.SetInode(inode)
					fileSystem.InodeTable[stat.Ino] = inode
				}
			}
		}
	}
	directory.FileList = append(directory.FileList, &file)
	return nil
}

func scanRegularFile(file *filesystem.RegularFile, fileSystem *FileSystem,
	parentName string) error {
	myPathName := path.Join(parentName, file.Name)
	f, err := os.Open(path.Join(fileSystem.rootDirectoryName, myPathName))
	if err != nil {
		return err
	}
	reader := fsrateio.NewReader(f, fileSystem.configuration.FsScanContext)
	hash := sha512.New()
	io.Copy(hash, reader)
	f.Close()
	copy(file.Inode().Hash[:], hash.Sum(nil))
	fileSystem.HashCount++
	return nil
}

func scanSymlink(symlink *filesystem.Symlink, fileSystem *FileSystem,
	parentName string) error {
	myPathName := path.Join(parentName, symlink.Name)
	target, err := os.Readlink(path.Join(fileSystem.rootDirectoryName,
		myPathName))
	if err != nil {
		return err
	}
	symlink.Inode().Symlink = target
	return nil
}

func scanFile(file *filesystem.File, fileSystem *FileSystem,
	parentName string) error {
	return nil
}
