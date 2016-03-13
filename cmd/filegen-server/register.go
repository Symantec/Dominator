package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filegen"
	"os"
	"path"
)

func registerSourceDirectory(manager *filegen.Manager, baseDir string,
	myPathName string) error {
	file, err := os.Open(path.Join(baseDir, myPathName))
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		filename := path.Join(myPathName, name)
		pathname := path.Join(baseDir, filename)
		fi, err := os.Lstat(pathname)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			if err := registerSourceDirectory(manager, baseDir,
				filename); err != nil {
				return err
			}
		} else if fi.Mode().IsRegular() {
			fmt.Println(filename, pathname)
			manager.RegisterFileForPath(filename, pathname)
		}
	}
	return nil
}
