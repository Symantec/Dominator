package main

import (
	"os"
	"path"
	"sync"
	"syscall"

	"github.com/Cloud-Foundations/Dominator/lib/concurrent"
)

type stateType struct {
	sync.Mutex
	processedInodes     map[uint64]struct{}
	directoriesToRemove []string
	concurrencyState    *concurrent.State
}

func walk(rootDirName, dirName, objectsDir string) error {
	var state stateType
	state.processedInodes = make(map[uint64]struct{})
	state.directoriesToRemove = make([]string, 0)
	state.concurrencyState = concurrent.NewState(0)
	if err := state.walk(rootDirName, dirName, objectsDir); err != nil {
		return err
	}
	if err := state.concurrencyState.Reap(); err != nil {
		return err
	}
	for _, dirname := range state.directoriesToRemove {
		if err := os.Remove(dirname); err != nil {
			return err
		}
	}
	return nil
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
				state.directoriesToRemove = append(state.directoriesToRemove,
					pathname)
			}
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFREG {
			err = state.concurrencyState.GoRun(func() error {
				return state.handleFile(pathname, stat.Ino, objectsDir)
			})
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
	state.Lock()
	if _, ok := state.processedInodes[inum]; ok {
		state.Unlock()
		return os.Remove(pathname)
	}
	state.processedInodes[inum] = struct{}{}
	state.Unlock()
	return convertToObject(pathname, objectsDir)
}
