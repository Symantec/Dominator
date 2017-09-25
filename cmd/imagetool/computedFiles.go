package main

import (
	"errors"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/util"
)

type computedFileType struct {
	Filename string
	Source   string
}

func spliceComputedFiles(fs *filesystem.FileSystem) error {
	if *computedFiles == "" {
		return nil
	}
	cfl, err := util.LoadComputedFiles(*computedFiles)
	if err != nil {
		return errors.New("cannot load computed files: " + err.Error())
	}
	if err := util.SpliceComputedFiles(fs, cfl); err != nil {
		return errors.New("cannot splice computed files: " + err.Error())
	}
	return nil
}
