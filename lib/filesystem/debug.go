package filesystem

import (
	"fmt"
	"io"
	"syscall"
)

func (fs *FileSystem) debugWrite(w io.Writer, prefix string) error {
	return fs.Directory.debugWrite(w, prefix)
}

func (directory *Directory) debugWrite(w io.Writer, prefix string) error {
	_, err := fmt.Fprintf(w, "%s%s\t%v %d %d\n", prefix, directory.Name,
		directory.Mode, directory.Uid, directory.Gid)
	if err != nil {
		return err
	}
	if len(directory.RegularFileList) > 0 {
		_, err = fmt.Fprintf(w, "%s Regular Files:\n", prefix)
		if err != nil {
			return err
		}
		for _, file := range directory.RegularFileList {
			err = file.DebugWrite(w, prefix+"  ")
			if err != nil {
				return err
			}
		}
	}
	if len(directory.SymlinkList) > 0 {
		_, err = fmt.Fprintf(w, "%s Symlinks:\n", prefix)
		if err != nil {
			return err
		}
		for _, symlink := range directory.SymlinkList {
			err = symlink.DebugWrite(w, prefix+"  ")
			if err != nil {
				return err
			}
		}
	}
	if len(directory.FileList) > 0 {
		_, err = fmt.Fprintf(w, "%s Files:\n", prefix)
		if err != nil {
			return err
		}
		for _, file := range directory.FileList {
			err = file.DebugWrite(w, prefix+"  ")
			if err != nil {
				return err
			}
		}
	}
	if len(directory.DirectoryList) > 0 {
		_, err = fmt.Fprintf(w, "%s Directories:\n", prefix)
		if err != nil {
			return err
		}
		for _, dir := range directory.DirectoryList {
			err = dir.DebugWrite(w, prefix+"  ")
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (file *RegularFile) debugWrite(w io.Writer, prefix string) error {
	inode := file.inode
	_, err := fmt.Fprintf(w, "%s%s\t%v %d %d %x\n", prefix, file.Name,
		inode.Mode, inode.Uid, inode.Gid, inode.Hash)
	if err != nil {
		return err
	}
	return nil
}

func (symlink *Symlink) debugWrite(w io.Writer, prefix string) error {
	inode := symlink.inode
	_, err := fmt.Fprintf(w, "%s%s\t%d %d %s\n", prefix, symlink.Name,
		inode.Uid, inode.Gid, inode.Symlink)
	if err != nil {
		return err
	}
	return nil
}

func (file *File) debugWrite(w io.Writer, prefix string) error {
	inode := file.inode
	var data string
	data = ""
	if inode.Mode&syscall.S_IFMT == syscall.S_IFBLK ||
		inode.Mode&syscall.S_IFMT == syscall.S_IFCHR {
		data = fmt.Sprintf(" %#x", inode.Rdev)
	}
	_, err := fmt.Fprintf(w, "%s%s\t%v %d %d%s\n", prefix, file.Name,
		inode.Mode, inode.Uid, inode.Gid, data)
	if err != nil {
		return err
	}
	return nil
}

func (mode FileMode) string() string {
	var buf [10]byte
	w := 1
	const rwx = "rwxrwxrwx"
	for i, c := range rwx {
		if mode&(1<<uint(9-1-i)) != 0 {
			buf[w] = byte(c)
		} else {
			buf[w] = '-'
		}
		w++
	}
	switch {
	case mode&syscall.S_IFMT == syscall.S_IFSOCK:
		buf[0] = 's'
	case mode&syscall.S_IFMT == syscall.S_IFLNK:
		buf[0] = 'l'
	case mode&syscall.S_IFMT == syscall.S_IFREG:
		buf[0] = '-'
	case mode&syscall.S_IFMT == syscall.S_IFBLK:
		buf[0] = 'b'
	case mode&syscall.S_IFMT == syscall.S_IFDIR:
		buf[0] = 'd'
	case mode&syscall.S_IFMT == syscall.S_IFCHR:
		buf[0] = 'c'
	case mode&syscall.S_IFMT == syscall.S_IFIFO:
		buf[0] = 'p'
	case mode&syscall.S_ISUID != 0:
		if mode&syscall.S_IXUSR == 0 {
			buf[3] = 'S'
		} else {
			buf[3] = 's'
		}
	case mode&syscall.S_ISGID != 0:
		if mode&syscall.S_IXGRP == 0 {
			buf[6] = 'S'
		} else {
			buf[6] = 's'
		}
	case mode&syscall.S_ISVTX != 0:
		if mode&syscall.S_IXOTH == 0 {
			buf[9] = 'T'
		} else {
			buf[9] = 't'
		}
	default:
		buf[0] = '?'
	}
	return string(buf[:])
}
