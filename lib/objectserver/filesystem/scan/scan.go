package scan

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Symantec/Dominator/lib/concurrent"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
)

func scanTree(baseDir string, registerFunc func(hash.Hash, uint64)) error {
	if fi, err := os.Stat(baseDir); err != nil {
		return fmt.Errorf("Cannot stat: %s: %s\n", baseDir, err)
	} else {
		if !fi.IsDir() {
			return fmt.Errorf("%s is not a directory\n", baseDir)
		}
	}
	state := concurrent.NewState(0)
	if err := scanDirectory(baseDir, "", state, registerFunc); err != nil {
		return err
	}
	if err := state.Reap(); err != nil {
		return err
	}
	return nil
}

func scanDirectory(baseDir string, subpath string, state *concurrent.State,
	registerFunc func(hash.Hash, uint64)) error {
	myPathName := filepath.Join(baseDir, subpath)
	file, err := os.Open(myPathName)
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		if len(name) > 0 && name[0] == '.' {
			continue // Skip hidden paths.
		}
		fullPathName := filepath.Join(myPathName, name)
		fi, err := os.Lstat(fullPathName)
		if err != nil {
			continue
		}
		filename := filepath.Join(subpath, name)
		if fi.IsDir() {
			if state == nil {
				err := scanDirectory(baseDir, filename, nil, registerFunc)
				if err != nil {
					return err
				}
			} else {
				// GoRun() cannot be used recursively, so limit concurrency to
				// the top level. It's also more efficient this way.
				if err := state.GoRun(func() error {
					return scanDirectory(baseDir, filename, nil, registerFunc)
				}); err != nil {
					return err
				}
			}
		} else {
			if fi.Size() < 1 {
				return fmt.Errorf("zero-length file: %s", fullPathName)
			}
			hashVal, err := objectcache.FilenameToHash(filename)
			if err != nil {
				return err
			}
			registerFunc(hashVal, uint64(fi.Size()))
		}
	}
	return nil
}
