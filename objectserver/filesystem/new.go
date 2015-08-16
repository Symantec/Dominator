package filesystem

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"os"
	"path"
)

func newObjectServer(baseDir string) (*ObjectServer, error) {
	fi, err := os.Stat(baseDir)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Cannot stat: %s\t%s\n", baseDir, err))
	}
	if !fi.IsDir() {
		return nil, errors.New(fmt.Sprintf("%s is not a directory\n", baseDir))
	}
	var objSrv ObjectServer
	objSrv.baseDir = baseDir
	objSrv.sizesMap = make(map[hash.Hash]uint64)
	err = scanDirectory(&objSrv, baseDir, "")
	if err != nil {
		return nil, err
	}
	return &objSrv, nil
}

func scanDirectory(objSrv *ObjectServer, baseDir string, subpath string) error {
	myPathName := path.Join(baseDir, subpath)
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
		fullPathName := path.Join(myPathName, name)
		fi, err := os.Lstat(fullPathName)
		if err != nil {
			continue
		}
		filename := path.Join(subpath, name)
		if fi.IsDir() {
			err = scanDirectory(objSrv, baseDir, filename)
			if err != nil {
				return err
			}
		} else {
			if fi.Size() < 1 {
				return errors.New(
					fmt.Sprintf("zero-length file: %s", fullPathName))
			}
			hash, err := objectcache.FilenameToHash(filename)
			if err != nil {
				return err
			}
			objSrv.sizesMap[hash] = uint64(fi.Size())
		}
	}
	return nil
}
