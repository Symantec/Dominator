package scanner

import (
	"bytes"
	"fmt"
	"io"
	"syscall"
)

func compare(left *FileSystem, right *FileSystem, logWriter io.Writer) bool {
	if len(left.InodeTable) != len(right.InodeTable) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d inodes\n",
				len(left.InodeTable), len(right.InodeTable))
		}
		return false
	}
	if !compareDirectories(&left.Directory, &right.Directory, logWriter) {
		return false
	}
	if len(left.ObjectCache) != len(right.ObjectCache) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d objects\n",
				len(left.ObjectCache), len(right.ObjectCache))
		}
		return false
	}
	return compareObjects(left.ObjectCache, right.ObjectCache, logWriter)
}

func compareDirectories(left, right *Directory, logWriter io.Writer) bool {
	if left.name != right.name {
		fmt.Fprintf(logWriter, "dirname: left vs. right: %s vs. %s\n",
			left.name, right.name)
		return false
	}
	if !compareInodes(left.inode, right.inode, logWriter) {
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
		fmt.Fprintf(logWriter, "left vs. right: %d vs. %d subdirs\n",
			len(left.DirectoryList), len(right.DirectoryList))
		return false
	}
	for index, leftFile := range left.FileList {
		if !compareFiles(leftFile, right.FileList[index], logWriter) {
			return false
		}
	}
	for index, leftDirectory := range left.DirectoryList {
		if !compareDirectories(leftDirectory, right.DirectoryList[index],
			logWriter) {
			return false
		}
	}
	return true
}

func compareFiles(left *File, right *File, logWriter io.Writer) bool {
	if left.name != right.name {
		fmt.Fprintf(logWriter, "filename: left vs. right: %s vs. %s\n",
			left.name, right.name)
		return false
	}
	if !compareInodes(left.inode, right.inode, logWriter) {
		return false
	}
	return true
}

func compareInodes(left *Inode, right *Inode, logWriter io.Writer) bool {
	if left.stat.Dev != right.stat.Dev {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "stat.Dev: left vs. right: %x vs. %x\n",
				left.stat.Dev, right.stat.Dev)
		}
		return false
	}
	if left.stat.Mode != right.stat.Mode {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "stat.Mode: left vs. right: %x vs. %x\n",
				left.stat.Mode, right.stat.Mode)
		}
		return false
	}
	if left.stat.Uid != right.stat.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "stat.Uid: left vs. right: %d vs. %d\n",
				left.stat.Uid, right.stat.Uid)
		}
		return false
	}
	if left.stat.Gid != right.stat.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "stat.Gid: left vs. right: %d vs. %d\n",
				left.stat.Gid, right.stat.Gid)
		}
		return false
	}
	if left.stat.Mode&syscall.S_IFMT != syscall.S_IFDIR {
		if left.stat.Size != right.stat.Size {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "stat.Size: left vs. right: %d vs. %d\n",
					left.stat.Size, right.stat.Size)
			}
			return false
		}
	}
	if left.stat.Mode&syscall.S_IFMT != syscall.S_IFDIR &&
		left.stat.Mode&syscall.S_IFMT != syscall.S_IFLNK {
		if left.stat.Mtim != right.stat.Mtim {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "stat.Mtim: left vs. right: %d vs. %d\n",
					left.stat.Mtim, right.stat.Mtim)
			}
			return false
		}
	}
	if left.stat.Mode&syscall.S_IFMT == syscall.S_IFREG {
		if bytes.Compare(left.hash, right.hash) != 0 {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "hash: left vs. right: %x vs. %x\n",
					left.hash, right.hash)
			}
			return false
		}
	}
	if left.stat.Mode&syscall.S_IFMT == syscall.S_IFBLK ||
		left.stat.Mode&syscall.S_IFMT == syscall.S_IFCHR {
		if left.stat.Rdev != right.stat.Rdev {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "stat.Rdev: left vs. right: %x vs. %x\n",
					left.stat.Rdev, right.stat.Rdev)
			}
			return false
		}
	}
	if left.stat.Mode&syscall.S_IFMT == syscall.S_IFLNK {
		if left.symlink != right.symlink {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "symlink: left vs. right: %s vs. %s\n",
					left.symlink, right.symlink)
			}
			return false
		}
	}
	return true
}

func compareObjects(left [][]byte, right [][]byte, logWriter io.Writer) bool {
	for index, leftHash := range left {
		if bytes.Compare(leftHash, right[index]) != 0 {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "hash: left vs. right: %x vs. %x\n",
					leftHash, right[index])
			}
			return false
		}
	}
	return true
}
