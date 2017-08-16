package fsutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"syscall"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
)

func copyToFile(destFilename string, perm os.FileMode, reader io.Reader,
	length uint64) error {
	tmpFilename := destFilename + "~"
	destFile, err := os.OpenFile(tmpFilename, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFilename)
	defer destFile.Close()
	var nCopied int64
	iLength := int64(length)
	if length < 1 {
		if _, err := io.Copy(destFile, reader); err != nil {
			return fmt.Errorf("error copying: %s", err)
		}
	} else {
		if nCopied, err = io.CopyN(destFile, reader, iLength); err != nil {
			return fmt.Errorf("error copying: %s", err)
		}
		if nCopied != iLength {
			return fmt.Errorf("expected length: %d, got: %d for: %s\n",
				length, nCopied, tmpFilename)
		}
	}
	return os.Rename(tmpFilename, destFilename)
}

func copyTree(destDir, sourceDir string,
	copyFunc func(destFilename, sourceFilename string,
		mode os.FileMode) error) error {
	file, err := os.Open(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		sourceFilename := path.Join(sourceDir, name)
		destFilename := path.Join(destDir, name)
		var stat syscall.Stat_t
		if err := syscall.Lstat(sourceFilename, &stat); err != nil {
			return errors.New(sourceFilename + ": " + err.Error())
		}
		switch stat.Mode & syscall.S_IFMT {
		case syscall.S_IFDIR:
			if err := os.Mkdir(destFilename, dirPerms); err != nil {
				if !os.IsExist(err) {
					return err
				}
			}
			err := copyTree(destFilename, sourceFilename, copyFunc)
			if err != nil {
				return err
			}
		case syscall.S_IFREG:
			err := copyFunc(destFilename, sourceFilename,
				os.FileMode(stat.Mode)&os.ModePerm)
			if err != nil {
				return err
			}
		case syscall.S_IFLNK:
			target, err := os.Readlink(sourceFilename)
			if err != nil {
				return errors.New(sourceFilename + ": " + err.Error())
			}
			if err := os.Symlink(target, destFilename); err != nil {
				return err
			}
		default:
			return errors.New("unsupported file type")
		}
	}
	return nil
}

func copyFile(destFilename, sourceFilename string, mode os.FileMode) error {
	if mode == 0 {
		var stat syscall.Stat_t
		if err := syscall.Stat(sourceFilename, &stat); err != nil {
			return errors.New(sourceFilename + ": " + err.Error())
		}
		mode = os.FileMode(stat.Mode & syscall.S_IFMT)
	}
	sourceFile, err := os.Open(sourceFilename)
	if err != nil {
		return errors.New(sourceFilename + ": " + err.Error())
	}
	defer sourceFile.Close()
	return copyToFile(destFilename, mode, sourceFile, 0)
}
