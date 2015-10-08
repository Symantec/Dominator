package scanner

import (
	"crypto/sha512"
	"errors"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/objectcache"
	"io"
	"os"
	"path"
	"sort"
	"syscall"
)

func makeRegularInode(stat *syscall.Stat_t) *filesystem.RegularInode {
	var inode filesystem.RegularInode
	inode.Mode = filesystem.FileMode(stat.Mode)
	inode.Uid = stat.Uid
	inode.Gid = stat.Gid
	inode.MtimeSeconds = stat.Mtim.Sec
	inode.MtimeNanoSeconds = int32(stat.Mtim.Nsec)
	inode.Size = uint64(stat.Size)
	return &inode
}

func makeSymlinkInode(stat *syscall.Stat_t) *filesystem.SymlinkInode {
	var inode filesystem.SymlinkInode
	inode.Uid = stat.Uid
	inode.Gid = stat.Gid
	return &inode
}

func makeInode(stat *syscall.Stat_t) *filesystem.Inode {
	var inode filesystem.Inode
	inode.Mode = filesystem.FileMode(stat.Mode)
	inode.Uid = stat.Uid
	inode.Gid = stat.Gid
	inode.MtimeSeconds = stat.Mtim.Sec
	inode.MtimeNanoSeconds = int32(stat.Mtim.Nsec)
	inode.Rdev = stat.Rdev
	return &inode
}

func scanFileSystem(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration, oldFS *FileSystem) (*FileSystem, error) {
	var fileSystem FileSystem
	fileSystem.configuration = configuration
	fileSystem.rootDirectoryName = rootDirectoryName
	fileSystem.cacheDirectoryName = cacheDirectoryName
	var stat syscall.Stat_t
	err := syscall.Lstat(rootDirectoryName, &stat)
	if err != nil {
		return nil, err
	}
	fileSystem.InodeTable = make(filesystem.InodeTable)
	fileSystem.dev = stat.Dev
	fileSystem.Mode = filesystem.FileMode(stat.Mode)
	fileSystem.Uid = stat.Uid
	fileSystem.Gid = stat.Gid
	fileSystem.InodeTable[stat.Ino] = &fileSystem.DirectoryInode
	fileSystem.DirectoryCount++
	var tmpInode filesystem.RegularInode
	if sha512.New().Size() != len(tmpInode.Hash) {
		return nil, errors.New("Incompatible hash size")
	}
	var oldDirectory *filesystem.DirectoryInode
	if oldFS != nil && oldFS.InodeTable != nil {
		oldDirectory = oldFS.FileSystem.InodeTable[stat.Ino].(*filesystem.DirectoryInode)
	}
	err = scanDirectory(&fileSystem.FileSystem.DirectoryInode, oldDirectory,
		&fileSystem, oldFS, "/")
	oldFS = nil
	oldDirectory = nil
	if err != nil {
		return nil, err
	}
	err = fileSystem.scanObjectCache()
	if err != nil {
		return nil, err
	}
	fileSystem.ComputeTotalDataBytes()
	return &fileSystem, nil
}

func (fs *FileSystem) scanObjectCache() error {
	if fs.cacheDirectoryName == "" {
		return nil
	}
	var err error
	fs.ObjectCache, err = objectcache.ScanObjectCache(fs.cacheDirectoryName)
	return err
}

func scanDirectory(directory, oldDirectory *filesystem.DirectoryInode,
	fileSystem, oldFS *FileSystem, myPathName string) error {
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
	entryList := make([]*filesystem.DirectoryEntry, 0, len(names))
	var copiedDirents int
	for _, name := range names {
		if directory == &fileSystem.DirectoryInode && name == ".subd" {
			continue
		}
		filename := path.Join(myPathName, name)
		if fileSystem.configuration.ScanFilter.Match(filename) {
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
		dirent := new(filesystem.DirectoryEntry)
		dirent.Name = name
		dirent.InodeNumber = stat.Ino
		var oldDirent *filesystem.DirectoryEntry
		if oldDirectory != nil {
			index := len(entryList)
			if len(oldDirectory.EntryList) > index &&
				oldDirectory.EntryList[index].Name == name {
				oldDirent = oldDirectory.EntryList[index]
			}
		}
		if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
			err = addDirectory(dirent, oldDirent, fileSystem, oldFS, myPathName,
				&stat)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFREG {
			err = addRegularFile(dirent, fileSystem, oldFS, myPathName, &stat)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFLNK {
			err = addSymlink(dirent, fileSystem, oldFS, myPathName, &stat)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFSOCK {
			continue
		} else {
			err = addFile(dirent, fileSystem, oldFS, &stat)
		}
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return err
		}
		if oldDirent != nil && *dirent == *oldDirent {
			dirent = oldDirent
			copiedDirents++
		}
		entryList = append(entryList, dirent)
	}
	if oldDirectory != nil && len(entryList) == copiedDirents {
		directory.EntryList = oldDirectory.EntryList
	} else {
		directory.EntryList = entryList
	}
	return nil
}

func addDirectory(dirent, oldDirent *filesystem.DirectoryEntry,
	fileSystem, oldFS *FileSystem,
	directoryPathName string, stat *syscall.Stat_t) error {
	myPathName := path.Join(directoryPathName, dirent.Name)
	if _, ok := fileSystem.InodeTable[stat.Ino]; ok {
		return errors.New("Hardlinked directory: " + myPathName)
	}
	var inode filesystem.DirectoryInode
	dirent.SetInode(&inode)
	inode.Mode = filesystem.FileMode(stat.Mode)
	inode.Uid = stat.Uid
	inode.Gid = stat.Gid
	fileSystem.InodeTable[stat.Ino] = &inode
	var oldInode *filesystem.DirectoryInode
	if oldDirent != nil {
		if oi, ok := oldDirent.Inode().(*filesystem.DirectoryInode); ok {
			oldInode = oi
		}
	}
	err := scanDirectory(&inode, oldInode, fileSystem, oldFS, myPathName)
	if err != nil {
		return err
	}
	fileSystem.DirectoryCount++
	return nil
}

func addRegularFile(dirent *filesystem.DirectoryEntry,
	fileSystem, oldFS *FileSystem,
	directoryPathName string, stat *syscall.Stat_t) error {
	if inode, ok := fileSystem.InodeTable[stat.Ino]; ok {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			dirent.SetInode(inode)
			return nil
		}
		return errors.New("Inode changed type")
	}
	inode := makeRegularInode(stat)
	if inode.Size > 0 {
		err := scanRegularInode(inode, fileSystem,
			path.Join(directoryPathName, dirent.Name))
		if err != nil {
			return err
		}
	}
	if oldFS != nil && oldFS.InodeTable != nil {
		if oldInode, found := oldFS.InodeTable[stat.Ino]; found {
			if oldInode, ok := oldInode.(*filesystem.RegularInode); ok {
				if filesystem.CompareRegularInodes(inode, oldInode, nil) {
					inode = oldInode
				}
			}
		}
	}
	dirent.SetInode(inode)
	fileSystem.InodeTable[stat.Ino] = inode
	return nil
}

func addSymlink(dirent *filesystem.DirectoryEntry,
	fileSystem, oldFS *FileSystem,
	directoryPathName string, stat *syscall.Stat_t) error {
	if inode, ok := fileSystem.InodeTable[stat.Ino]; ok {
		if inode, ok := inode.(*filesystem.SymlinkInode); ok {
			dirent.SetInode(inode)
			return nil
		}
		return errors.New("Inode changed type")
	}
	inode := makeSymlinkInode(stat)
	err := scanSymlinkInode(inode, fileSystem,
		path.Join(directoryPathName, dirent.Name))
	if err != nil {
		return err
	}
	if oldFS != nil && oldFS.InodeTable != nil {
		if oldInode, found := oldFS.InodeTable[stat.Ino]; found {
			if oldInode, ok := oldInode.(*filesystem.SymlinkInode); ok {
				if filesystem.CompareSymlinkInodes(inode, oldInode, nil) {
					inode = oldInode
				}
			}
		}
	}
	dirent.SetInode(inode)
	fileSystem.InodeTable[stat.Ino] = inode
	return nil
}

func addFile(dirent *filesystem.DirectoryEntry, fileSystem, oldFS *FileSystem,
	stat *syscall.Stat_t) error {
	if inode, ok := fileSystem.InodeTable[stat.Ino]; ok {
		if inode, ok := inode.(*filesystem.Inode); ok {
			dirent.SetInode(inode)
			return nil
		}
		return errors.New("Inode changed type")
	}
	inode := makeInode(stat)
	if oldFS != nil && oldFS.InodeTable != nil {
		if oldInode, found := oldFS.InodeTable[stat.Ino]; found {
			if oldInode, ok := oldInode.(*filesystem.Inode); ok {
				if filesystem.CompareInodes(inode, oldInode, nil) {
					inode = oldInode
				}
			}
		}
	}
	dirent.SetInode(inode)
	fileSystem.InodeTable[stat.Ino] = inode
	return nil
}

func scanRegularInode(inode *filesystem.RegularInode, fileSystem *FileSystem,
	myPathName string) error {
	f, err := os.Open(path.Join(fileSystem.rootDirectoryName, myPathName))
	if err != nil {
		return err
	}
	reader := fileSystem.configuration.FsScanContext.NewReader(f)
	hash := sha512.New()
	io.Copy(hash, reader)
	f.Close()
	copy(inode.Hash[:], hash.Sum(nil))
	return nil
}

func scanSymlinkInode(inode *filesystem.SymlinkInode, fileSystem *FileSystem,
	myPathName string) error {
	target, err := os.Readlink(path.Join(fileSystem.rootDirectoryName,
		myPathName))
	if err != nil {
		return err
	}
	inode.Symlink = target
	return nil
}
