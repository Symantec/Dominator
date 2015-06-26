package scanner

import (
	"bytes"
	"fmt"
	"syscall"
)

func compare(left *FileSystem, right *FileSystem, verbose bool) bool {
	if len(left.InodeTable) != len(right.InodeTable) {
		fmt.Printf("left vs. right: %d vs. %d inodes\n",
			len(left.InodeTable), len(right.InodeTable))
		return false
	}
	return compareDirectories(&left.Directory, &right.Directory, verbose)
}

func compareDirectories(left *Directory, right *Directory, verbose bool) bool {
	if left.name != right.name {
		fmt.Printf("dirname: left vs. right: %s vs. %s\n",
			left.name, right.name)
		return false
	}
	if !compareInodes(left.inode, right.inode, verbose) {
		return false
	}
	if len(left.FileList) != len(right.FileList) {
		fmt.Printf("left vs. right: %d vs. %d files\n",
			len(left.FileList), len(right.FileList))
		return false
	}
	if len(left.DirectoryList) != len(right.DirectoryList) {
		fmt.Printf("left vs. right: %d vs. %d subdirs\n",
			len(left.DirectoryList), len(right.DirectoryList))
		return false
	}
	for index := 0; index < len(left.FileList); index++ {
		if !compareFiles(left.FileList[index], right.FileList[index], verbose) {
			return false
		}
	}
	for index := 0; index < len(left.DirectoryList); index++ {
		if !compareDirectories(left.DirectoryList[index],
			right.DirectoryList[index], verbose) {
			return false
		}
	}
	return true
}

func compareFiles(left *File, right *File, verbose bool) bool {
	if left.name != right.name {
		fmt.Printf("filename: left vs. right: %s vs. %s\n",
			left.name, right.name)
		return false
	}
	if !compareInodes(left.inode, right.inode, verbose) {
		return false
	}
	return true
}

func compareInodes(left *Inode, right *Inode, verbose bool) bool {
	if left.stat.Dev != right.stat.Dev {
		if verbose {
			fmt.Printf("stat.Dev: left vs. right: %x vs. %x\n",
				left.stat.Dev, right.stat.Dev)
		}
		return false
	}
	if left.stat.Mode != right.stat.Mode {
		if verbose {
			fmt.Printf("stat.Mode: left vs. right: %x vs. %x\n",
				left.stat.Mode, right.stat.Mode)
		}
		return false
	}
	if left.stat.Uid != right.stat.Uid {
		if verbose {
			fmt.Printf("stat.Uid: left vs. right: %d vs. %d\n",
				left.stat.Uid, right.stat.Uid)
		}
		return false
	}
	if left.stat.Gid != right.stat.Gid {
		if verbose {
			fmt.Printf("stat.Gid: left vs. right: %d vs. %d\n",
				left.stat.Gid, right.stat.Gid)
		}
		return false
	}
	if left.stat.Mode&syscall.S_IFMT != syscall.S_IFDIR {
		if left.stat.Size != right.stat.Size {
			if verbose {
				fmt.Printf("stat.Size: left vs. right: %d vs. %d\n",
					left.stat.Size, right.stat.Size)
			}
			return false
		}
	}
	if left.stat.Mode&syscall.S_IFMT != syscall.S_IFDIR &&
		left.stat.Mode&syscall.S_IFMT != syscall.S_IFLNK {
		if left.stat.Mtim != right.stat.Mtim {
			if verbose {
				fmt.Printf("stat.Mtim: left vs. right: %d vs. %d\n",
					left.stat.Mtim, right.stat.Mtim)
			}
			return false
		}
	}
	if left.stat.Mode&syscall.S_IFMT == syscall.S_IFREG {
		if bytes.Compare(left.hash, right.hash) != 0 {
			if verbose {
				fmt.Printf("hash: left vs. right: %x vs. %x\n",
					left.hash, right.hash)
			}
			return false
		}
	}
	if left.stat.Mode&syscall.S_IFMT == syscall.S_IFBLK ||
		left.stat.Mode&syscall.S_IFMT == syscall.S_IFCHR {
		if left.stat.Rdev != right.stat.Rdev {
			if verbose {
				fmt.Printf("stat.Rdev: left vs. right: %x vs. %x\n",
					left.stat.Rdev, right.stat.Rdev)
			}
			return false
		}
	}
	if left.stat.Mode&syscall.S_IFMT == syscall.S_IFLNK {
		if left.symlink != right.symlink {
			if verbose {
				fmt.Printf("symlink: left vs. right: %s vs. %s\n",
					left.symlink, right.symlink)
			}
			return false
		}
	}
	return true
}
