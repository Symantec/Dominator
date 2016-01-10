package main

import (
	"os"
	"path"
	"syscall"
)

type stateType struct {
	processedInodes map[uint64]struct{}
}

func walk(rootDirName, dirName, objectsDir string) error {
	var state stateType
	state.processedInodes = make(map[uint64]struct{})
	return state.walk(rootDirName, dirName, objectsDir)
}

func (state *stateType) walk(rootDirName, dirName, objectsDir string) error {
	file, err := os.Open(path.Join(rootDirName, dirName))
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		if dirName == "/" && name == ".subd" {
			continue
		}
		filename := path.Join(dirName, name)
		pathname := path.Join(rootDirName, filename)
		var stat syscall.Stat_t
		err := syscall.Lstat(pathname, &stat)
		if err != nil {
			return err
		}
		if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
			err = state.walk(rootDirName, filename, objectsDir)
			if err == nil {
				err = os.Remove(pathname)
			}
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFREG {
			err = state.handleFile(pathname, stat.Ino, objectsDir)
		} else {
			err = os.RemoveAll(pathname)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (state *stateType) handleFile(pathname string, inum uint64,
	objectsDir string) error {
	if _, ok := state.processedInodes[inum]; ok {
		return os.Remove(pathname)
	}
	state.processedInodes[inum] = struct{}{}
	return convertToObject(pathname, objectsDir)
}
