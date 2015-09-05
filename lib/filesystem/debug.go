package filesystem

import (
	"fmt"
	"io"
)

func (fs *FileSystem) debugWrite(w io.Writer, prefix string) error {
	return fs.Directory.debugWrite(w, prefix)
}

func (directory *Directory) debugWrite(w io.Writer, prefix string) error {
	_, err := fmt.Fprintf(w, "%s%s\t%o %d %d\n", prefix, directory.Name,
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
	_, err := fmt.Fprintf(w, "%s%s\t%o %d %d %x\n", prefix, file.Name,
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
	_, err := fmt.Fprintf(w, "%s%s\t%o %d %d %s\n", prefix, file.Name,
		inode.Mode, inode.Uid, inode.Gid, data)
	if err != nil {
		return err
	}
	return nil
}
