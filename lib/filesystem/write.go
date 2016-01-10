package filesystem

import (
	"errors"
	"github.com/Symantec/Dominator/lib/fsutil"
	"os"
	"syscall"
	"time"
)

var modePerm FileMode = syscall.S_IRWXU | syscall.S_IRWXG | syscall.S_IRWXO

func forceWriteMetadata(inode GenericInode, name string) error {
	err := inode.WriteMetadata(name)
	if err == nil {
		return nil
	}
	if os.IsPermission(err) {
		// Blindly attempt to remove immutable attributes.
		fsutil.MakeMutable(name)
	}
	return inode.WriteMetadata(name)
}

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
	if inode.make(name) != nil {
		fsutil.ForceRemoveAll(name)
		if err := inode.make(name); err != nil {
			return err
		}
	}
	return inode.writeMetadata(name)
}

func (inode *SymlinkInode) make(name string) error {
	return os.Symlink(inode.Symlink, name)
}

func (inode *SymlinkInode) writeMetadata(name string) error {
	return os.Lchown(name, int(inode.Uid), int(inode.Gid))
}

func (inode *SpecialInode) write(name string) error {
	if inode.make(name) != nil {
		fsutil.ForceRemoveAll(name)
		if err := inode.make(name); err != nil {
			return err
		}
	}
	return inode.writeMetadata(name)
}

func (inode *SpecialInode) make(name string) error {
	if inode.Mode&syscall.S_IFBLK != 0 || inode.Mode&syscall.S_IFCHR != 0 {
		return syscall.Mknod(name, uint32(inode.Mode), int(inode.Rdev))
	} else if inode.Mode&syscall.S_IFIFO != 0 {
		return syscall.Mkfifo(name, uint32(inode.Mode))
	} else {
		return errors.New("unsupported mode")
	}
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
	if inode.make(name) != nil {
		fsutil.ForceRemoveAll(name)
		if err := inode.make(name); err != nil {
			return err
		}
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

func (inode *DirectoryInode) make(name string) error {
	return syscall.Mkdir(name, uint32(inode.Mode))
}

func (inode *DirectoryInode) writeMetadata(name string) error {
	if err := os.Lchown(name, int(inode.Uid), int(inode.Gid)); err != nil {
		return err
	}
	return syscall.Chmod(name, uint32(inode.Mode))
}
