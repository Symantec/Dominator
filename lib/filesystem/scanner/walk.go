package scanner

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/fsrateio"
	"io"
	"os"
	"path"
	"runtime"
	"sort"
	"syscall"
)

var myCountGC int

func myGC() {
	if myCountGC > 1000 {
		runtime.GC()
		myCountGC = 0
	}
	myCountGC++
}

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

func makeSpecialInode(stat *syscall.Stat_t) *filesystem.SpecialInode {
	var inode filesystem.SpecialInode
	inode.Mode = filesystem.FileMode(stat.Mode)
	inode.Uid = stat.Uid
	inode.Gid = stat.Gid
	inode.MtimeSeconds = stat.Mtim.Sec
	inode.MtimeNanoSeconds = int32(stat.Mtim.Nsec)
	inode.Rdev = stat.Rdev
	return &inode
}

func scanFileSystem(rootDirectoryName string,
	fsScanContext *fsrateio.ReaderContext, scanFilter *filter.Filter,
	checkScanDisableRequest func() bool, oldFS *FileSystem) (
	*FileSystem, error) {
	var fileSystem FileSystem
	fileSystem.rootDirectoryName = rootDirectoryName
	fileSystem.fsScanContext = fsScanContext
	fileSystem.scanFilter = scanFilter
	fileSystem.checkScanDisableRequest = checkScanDisableRequest
	var stat syscall.Stat_t
	if err := syscall.Lstat(rootDirectoryName, &stat); err != nil {
		return nil, err
	}
	fileSystem.InodeTable = make(filesystem.InodeTable)
	fileSystem.dev = stat.Dev
	fileSystem.inodeNumber = stat.Ino
	fileSystem.Mode = filesystem.FileMode(stat.Mode)
	fileSystem.Uid = stat.Uid
	fileSystem.Gid = stat.Gid
	fileSystem.DirectoryCount++
	var tmpInode filesystem.RegularInode
	if sha512.New().Size() != len(tmpInode.Hash) {
		return nil, errors.New("incompatible hash size")
	}
	var oldDirectory *filesystem.DirectoryInode
	if oldFS != nil && oldFS.InodeTable != nil {
		oldDirectory = &oldFS.DirectoryInode
	}
	err, _ := scanDirectory(&fileSystem.FileSystem.DirectoryInode, oldDirectory,
		&fileSystem, oldFS, "/")
	oldFS = nil
	oldDirectory = nil
	if err != nil {
		return nil, err
	}
	fileSystem.ComputeTotalDataBytes()
	if err = fileSystem.RebuildInodePointers(); err != nil {
		panic(err)
	}
	return &fileSystem, nil
}

func scanDirectory(directory, oldDirectory *filesystem.DirectoryInode,
	fileSystem, oldFS *FileSystem, myPathName string) (error, bool) {
	file, err := os.Open(path.Join(fileSystem.rootDirectoryName, myPathName))
	if err != nil {
		return err, false
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err, false
	}
	sort.Strings(names)
	entryList := make([]*filesystem.DirectoryEntry, 0, len(names))
	var copiedDirents int
	for _, name := range names {
		if directory == &fileSystem.DirectoryInode && name == ".subd" {
			continue
		}
		filename := path.Join(myPathName, name)
		if fileSystem.scanFilter != nil &&
			fileSystem.scanFilter.Match(filename) {
			continue
		}
		var stat syscall.Stat_t
		err := syscall.Lstat(path.Join(fileSystem.rootDirectoryName, filename),
			&stat)
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return err, false
		}
		if stat.Dev != fileSystem.dev {
			continue
		}
		if fileSystem.checkScanDisableRequest != nil &&
			fileSystem.checkScanDisableRequest() {
			return errors.New("DisableScan"), false
		}
		myGC()
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
			err = addSpecialFile(dirent, fileSystem, oldFS, &stat)
		}
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return err, false
		}
		if oldDirent != nil && *dirent == *oldDirent {
			dirent = oldDirent
			copiedDirents++
		}
		entryList = append(entryList, dirent)
	}
	if oldDirectory != nil && len(entryList) == copiedDirents &&
		len(entryList) == len(oldDirectory.EntryList) {
		directory.EntryList = oldDirectory.EntryList
		return nil, true
	} else {
		directory.EntryList = entryList
		return nil, false
	}
}

func addDirectory(dirent, oldDirent *filesystem.DirectoryEntry,
	fileSystem, oldFS *FileSystem,
	directoryPathName string, stat *syscall.Stat_t) error {
	myPathName := path.Join(directoryPathName, dirent.Name)
	if stat.Ino == fileSystem.inodeNumber {
		return errors.New("recursive directory: " + myPathName)
	}
	if _, ok := fileSystem.InodeTable[stat.Ino]; ok {
		return errors.New("hardlinked directory: " + myPathName)
	}
	inode := new(filesystem.DirectoryInode)
	dirent.SetInode(inode)
	fileSystem.InodeTable[stat.Ino] = inode
	inode.Mode = filesystem.FileMode(stat.Mode)
	inode.Uid = stat.Uid
	inode.Gid = stat.Gid
	var oldInode *filesystem.DirectoryInode
	if oldDirent != nil {
		if oi, ok := oldDirent.Inode().(*filesystem.DirectoryInode); ok {
			oldInode = oi
		}
	}
	err, copied := scanDirectory(inode, oldInode, fileSystem, oldFS, myPathName)
	if err != nil {
		return err
	}
	if copied && filesystem.CompareDirectoriesMetadata(inode, oldInode, nil) {
		dirent.SetInode(oldInode)
		fileSystem.InodeTable[stat.Ino] = oldInode
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
		return errors.New("inode changed type: " + dirent.Name)
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
		return errors.New("inode changed type: " + dirent.Name)
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

func addSpecialFile(dirent *filesystem.DirectoryEntry,
	fileSystem, oldFS *FileSystem, stat *syscall.Stat_t) error {
	if inode, ok := fileSystem.InodeTable[stat.Ino]; ok {
		if inode, ok := inode.(*filesystem.SpecialInode); ok {
			dirent.SetInode(inode)
			return nil
		}
		return errors.New("inode changed type: " + dirent.Name)
	}
	inode := makeSpecialInode(stat)
	if oldFS != nil && oldFS.InodeTable != nil {
		if oldInode, found := oldFS.InodeTable[stat.Ino]; found {
			if oldInode, ok := oldInode.(*filesystem.SpecialInode); ok {
				if filesystem.CompareSpecialInodes(inode, oldInode, nil) {
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
	defer f.Close()
	reader := io.Reader(f)
	if fileSystem.fsScanContext != nil {
		reader = fileSystem.fsScanContext.NewReader(f)
	}
	hash := sha512.New()
	nCopied, err := io.Copy(hash, reader)
	if err != nil {
		return err
	}
	if nCopied != int64(inode.Size) {
		return fmt.Errorf(
			"scanRegularInode(%s): read: %d, expected: %d bytes",
			myPathName, nCopied, inode.Size)
	}
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
