package scanner

import (
	"fmt"
	"io"
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
	_, err := fmt.Fprintf(w, "%s%s\t%x\n", prefix, file.Name, file.inode.Hash)
	if err != nil {
		return err
	}
	return nil
}

func (symlink *Symlink) debugWrite(w io.Writer, prefix string) error {
	_, err := fmt.Fprintf(w, "%s%s\t%s\n", prefix, symlink.Name,
		symlink.inode.Symlink)
	if err != nil {
		return err
	}
	return nil
}

func (file *File) debugWrite(w io.Writer, prefix string) error {
	var data string
	data = ""
	_, err := fmt.Fprintf(w, "%s%s\t%s\n", prefix, file.Name, data)
	if err != nil {
		return err
	}
	return nil
}
