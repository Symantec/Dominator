package util

import (
	"os"
	"path/filepath"

	"github.com/Symantec/Dominator/lib/filter"
)

func deletedFilteredFiles(rootDir string, filt *filter.Filter) error {
	startPos := len(rootDir)
	return filepath.Walk(rootDir,
		func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filt.Match(path[startPos:]) {
				if err := os.RemoveAll(path); err != nil {
					return err
				}
				if fi.IsDir() {
					return filepath.SkipDir
				}
			}
			return nil
		})
}
