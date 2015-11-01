package filesystem

import (
	"errors"
	"os"
	"syscall"
	"time"
)

var modePerm FileMode = syscall.S_IRWXU | syscall.S_IRWXG | syscall.S_IRWXO

func (inode *RegularInode) writeMetadata(name string) error {
	if err := os.Lchown(name, int(inode.Uid), int(inode.Gid)); err != nil {
		return err
	}
	if err := syscall.Chmod(name, uint32(inode.Mode)); err != nil {
		return err
	}
	t := time.Unix(inode.MtimeSeconds, int64(inode.MtimeNanoSeconds))
	return os.Chtimes(name, t, t)
}

func (inode *SymlinkInode) write(name string) error {
	if err := os.Symlink(inode.Symlink, name); err != nil {
		return err
	}
	return inode.writeMetadata(name)
}

func (inode *SymlinkInode) writeMetadata(name string) error {
	return os.Lchown(name, int(inode.Uid), int(inode.Gid))
}

func (inode *SpecialInode) write(name string) error {
	var err error
	if inode.Mode&syscall.S_IFBLK != 0 || inode.Mode&syscall.S_IFCHR != 0 {
		err = syscall.Mknod(name, uint32(inode.Mode), int(inode.Rdev))
	} else if inode.Mode&syscall.S_IFIFO != 0 {
		err = syscall.Mkfifo(name, uint32(inode.Mode))
	} else {
		return errors.New("unsupported mode")
	}
	if err != nil {
		return err
	}
	if err := os.Lchown(name, int(inode.Uid), int(inode.Gid)); err != nil {
		return err
	}
	t := time.Unix(inode.MtimeSeconds, int64(inode.MtimeNanoSeconds))
	if err := os.Chtimes(name, t, t); err != nil {
		return err
	}
	return nil
}

func (inode *SpecialInode) writeMetadata(name string) error {
	if err := os.Lchown(name, int(inode.Uid), int(inode.Gid)); err != nil {
		return err
	}
	if err := syscall.Chmod(name, uint32(inode.Mode)); err != nil {
		return err
	}
	t := time.Unix(inode.MtimeSeconds, int64(inode.MtimeNanoSeconds))
	return os.Chtimes(name, t, t)
}

func (inode *DirectoryInode) write(name string) error {
	if err := syscall.Mkdir(name, uint32(inode.Mode)); err != nil {
		return err
	}
	if err := os.Lchown(name, int(inode.Uid), int(inode.Gid)); err != nil {
		return err
	}
	if inode.Mode & ^modePerm != syscall.S_IFDIR {
		if err := syscall.Chmod(name, uint32(inode.Mode)); err != nil {
			return err
		}
	}
	return nil
}

func (inode *DirectoryInode) writeMetadata(name string) error {
	if err := os.Lchown(name, int(inode.Uid), int(inode.Gid)); err != nil {
		return err
	}
	return syscall.Chmod(name, uint32(inode.Mode))
}
