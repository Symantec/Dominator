package scanner

import (
	"fmt"
	"io"
	"syscall"
)

func (fs *FileSystem) debugWrite(w io.Writer, prefix string) error {
	_, err := fmt.Fprint(w, fs)
	if err != nil {
		return err
	}
	return fs.Directory.debugWrite(w, prefix)
}

func (directory *Directory) debugWrite(w io.Writer, prefix string) error {
	_, err := fmt.Fprintf(w, "%s%s\n", prefix, directory.Name)
	if err != nil {
		return err
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

func (file *File) debugWrite(w io.Writer, prefix string) error {
	var data string
	inode := file.inode
	if inode.Mode&syscall.S_IFMT == syscall.S_IFREG {
		data = fmt.Sprintf("%x", inode.Hash)
	} else if len(inode.Symlink) > 0 {
		data = inode.Symlink
	} else {
		data = ""
	}
	_, err := fmt.Fprintf(w, "%s%s\t%s\n", prefix, file.Name, data)
	if err != nil {
		return err
	}
	return nil
}
