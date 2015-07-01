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
	if left.Name != right.Name {
		fmt.Fprintf(logWriter, "dirname: left vs. right: %s vs. %s\n",
			left.Name, right.Name)
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
	if left.Name != right.Name {
		fmt.Fprintf(logWriter, "filename: left vs. right: %s vs. %s\n",
			left.Name, right.Name)
		return false
	}
	if !compareInodes(left.inode, right.inode, logWriter) {
		return false
	}
	return true
}

func compareInodes(left *Inode, right *Inode, logWriter io.Writer) bool {
	if left.Stat.Dev != right.Stat.Dev {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "stat.Dev: left vs. right: %x vs. %x\n",
				left.Stat.Dev, right.Stat.Dev)
		}
		return false
	}
	if left.Stat.Mode != right.Stat.Mode {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "stat.Mode: left vs. right: %x vs. %x\n",
				left.Stat.Mode, right.Stat.Mode)
		}
		return false
	}
	if left.Stat.Uid != right.Stat.Uid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "stat.Uid: left vs. right: %d vs. %d\n",
				left.Stat.Uid, right.Stat.Uid)
		}
		return false
	}
	if left.Stat.Gid != right.Stat.Gid {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "stat.Gid: left vs. right: %d vs. %d\n",
				left.Stat.Gid, right.Stat.Gid)
		}
		return false
	}
	if left.Stat.Mode&syscall.S_IFMT != syscall.S_IFDIR {
		if left.Stat.Size != right.Stat.Size {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "stat.Size: left vs. right: %d vs. %d\n",
					left.Stat.Size, right.Stat.Size)
			}
			return false
		}
	}
	if left.Stat.Mode&syscall.S_IFMT != syscall.S_IFDIR &&
		left.Stat.Mode&syscall.S_IFMT != syscall.S_IFLNK {
		if left.Stat.Mtim != right.Stat.Mtim {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "stat.Mtim: left vs. right: %d vs. %d\n",
					left.Stat.Mtim, right.Stat.Mtim)
			}
			return false
		}
	}
	if left.Stat.Mode&syscall.S_IFMT == syscall.S_IFREG {
		if bytes.Compare(left.Hash, right.Hash) != 0 {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "hash: left vs. right: %x vs. %x\n",
					left.Hash, right.Hash)
			}
			return false
		}
	}
	if left.Stat.Mode&syscall.S_IFMT == syscall.S_IFBLK ||
		left.Stat.Mode&syscall.S_IFMT == syscall.S_IFCHR {
		if left.Stat.Rdev != right.Stat.Rdev {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "stat.Rdev: left vs. right: %x vs. %x\n",
					left.Stat.Rdev, right.Stat.Rdev)
			}
			return false
		}
	}
	if left.Stat.Mode&syscall.S_IFMT == syscall.S_IFLNK {
		if left.Symlink != right.Symlink {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "symlink: left vs. right: %s vs. %s\n",
					left.Symlink, right.Symlink)
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
