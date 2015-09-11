package filesystem

import (
	"bytes"
	"fmt"
	"io"
	"syscall"
)

func compareFileSystems(left, right *FileSystem, logWriter io.Writer) bool {
	if len(left.RegularInodeTable) != len(right.RegularInodeTable) {
		if logWriter != nil {
			fmt.Fprintf(logWriter,
				"left vs. right: %d vs. %d regular file inodes\n",
				len(left.RegularInodeTable), len(right.RegularInodeTable))
		}
		return false
	}
	if len(left.SymlinkInodeTable) != len(right.SymlinkInodeTable) {
		if logWriter != nil {
			fmt.Fprintf(logWriter,
				"left vs. right: %d vs. %d symlink inodes\n",
				len(left.SymlinkInodeTable), len(right.SymlinkInodeTable))
		}
		return false
	}
	if len(left.InodeTable) != len(right.InodeTable) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d inodes\n",
				len(left.InodeTable), len(right.InodeTable))
		}
		return false
	}
	return compareDirectories(&left.Directory, &right.Directory, logWriter)
}

func compareDirectories(left, right *Directory, logWriter io.Writer) bool {
	if left.Name != right.Name {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "dirname: left vs. right: %s vs. %s\n",
				left.Name, right.Name)
		}
		return false
	}
	if left.Mode != right.Mode {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mode: left vs. right: %o vs. %o\n",
				left.Mode, right.Mode)
		}
		return false
	}
	if left.Uid != right.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Uid: left vs. right: %d vs. %d\n",
				left.Uid, right.Uid)
		}
		return false
	}
	if left.Gid != right.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %d vs. %d\n",
				left.Gid, right.Gid)
		}
		return false
	}
	if len(left.RegularFileList) != len(right.RegularFileList) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d regular files\n",
				len(left.RegularFileList), len(right.RegularFileList))
		}
		return false
	}
	if len(left.SymlinkList) != len(right.SymlinkList) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d symlinks\n",
				len(left.SymlinkList), len(right.SymlinkList))
		}
		return false
	}
	if len(left.FileList) != len(right.FileList) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d files\n",
				len(left.FileList), len(right.FileList))
		}
		return false
	}
	if len(left.DirectoryList) != len(right.DirectoryList) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d subdirs\n",
				len(left.DirectoryList), len(right.DirectoryList))
		}
		return false
	}
	for index, leftEntry := range left.RegularFileList {
		if !compareRegularFiles(leftEntry, right.RegularFileList[index],
			logWriter) {
			return false
		}
	}
	for index, leftEntry := range left.SymlinkList {
		if !compareSymlinks(leftEntry, right.SymlinkList[index], logWriter) {
			return false
		}
	}
	for index, leftEntry := range left.FileList {
		if !compareFiles(leftEntry, right.FileList[index], logWriter) {
			return false
		}
	}
	for index, leftEntry := range left.DirectoryList {
		if !compareDirectories(leftEntry, right.DirectoryList[index],
			logWriter) {
			return false
		}
	}
	return true
}

func compareRegularFiles(left, right *RegularFile, logWriter io.Writer) bool {
	if left.Name != right.Name {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "filename: left vs. right: %s vs. %s\n",
				left.Name, right.Name)
		}
		return false
	}
	return compareRegularInodes(left.inode, right.inode, logWriter)
}

func compareRegularInodes(left, right *RegularInode, logWriter io.Writer) bool {
	if !compareRegularInodesMetadata(left, right, logWriter) {
		return false
	}
	return compareRegularInodesData(left, right, logWriter)
}

func compareRegularInodesMetadata(left, right *RegularInode,
	logWriter io.Writer) bool {
	if left.Mode != right.Mode {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mode: left vs. right: %o vs. %o\n",
				left.Mode, right.Mode)
		}
		return false
	}
	if left.Uid != right.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Uid: left vs. right: %d vs. %d\n",
				left.Uid, right.Uid)
		}
		return false
	}
	if left.Gid != right.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %d vs. %d\n",
				left.Gid, right.Gid)
		}
		return false
	}
	var leftMtime, rightMtime syscall.Timespec
	leftMtime.Sec = left.MtimeSeconds
	leftMtime.Nsec = int64(left.MtimeNanoSeconds)
	rightMtime.Sec = right.MtimeSeconds
	rightMtime.Nsec = int64(right.MtimeNanoSeconds)
	if leftMtime != rightMtime {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mtime: left vs. right: %v vs. %v\n",
				leftMtime, rightMtime)
		}
		return false
	}
	return true
}

func compareRegularInodesData(left, right *RegularInode,
	logWriter io.Writer) bool {
	if left.Size != right.Size {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Size: left vs. right: %d vs. %d\n",
				left.Size, right.Size)
		}
		return false
	}
	if left.Size > 0 {
		if bytes.Compare(left.Hash[:], right.Hash[:]) != 0 {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "hash: left vs. right: %x vs. %x\n",
					left.Hash, right.Hash)
			}
			return false
		}
	}
	return true
}

func compareSymlinks(left, right *Symlink, logWriter io.Writer) bool {
	if left.Name != right.Name {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "filename: left vs. right: %s vs. %s\n",
				left.Name, right.Name)
		}
		return false
	}
	return compareSymlinkInodes(left.inode, right.inode, logWriter)
}

func compareSymlinkInodes(left, right *SymlinkInode, logWriter io.Writer) bool {
	if left.Uid != right.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Uid: left vs. right: %d vs. %d\n",
				left.Uid, right.Uid)
		}
		return false
	}
	if left.Gid != right.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %d vs. %d\n",
				left.Gid, right.Gid)
		}
		return false
	}
	if left.Symlink != right.Symlink {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "symlink: left vs. right: %s vs. %s\n",
				left.Symlink, right.Symlink)
		}
		return false
	}
	return true
}

func compareFiles(left, right *File, logWriter io.Writer) bool {
	if left.Name != right.Name {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "filename: left vs. right: %s vs. %s\n",
				left.Name, right.Name)
		}
		return false
	}
	return compareInodes(left.inode, right.inode, logWriter)
}

func compareInodes(left, right *Inode, logWriter io.Writer) bool {
	if left.Mode != right.Mode {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mode: left vs. right: %o vs. %o\n",
				left.Mode, right.Mode)
		}
		return false
	}
	if left.Uid != right.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Uid: left vs. right: %d vs. %d\n",
				left.Uid, right.Uid)
		}
		return false
	}
	if left.Gid != right.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Gid: left vs. right: %d vs. %d\n",
				left.Gid, right.Gid)
		}
		return false
	}
	var leftMtime, rightMtime syscall.Timespec
	leftMtime.Sec = left.MtimeSeconds
	leftMtime.Nsec = int64(left.MtimeNanoSeconds)
	rightMtime.Sec = right.MtimeSeconds
	rightMtime.Nsec = int64(right.MtimeNanoSeconds)
	if leftMtime != rightMtime {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "Mtime: left vs. right: %v vs. %v\n",
				leftMtime, rightMtime)
		}
		return false
	}
	if left.Mode&syscall.S_IFMT == syscall.S_IFBLK ||
		left.Mode&syscall.S_IFMT == syscall.S_IFCHR {
		if left.Rdev != right.Rdev {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "Rdev: left vs. right: %#x vs. %#x\n",
					left.Rdev, right.Rdev)
			}
			return false
		}
	}
	return true
}
