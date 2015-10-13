package scanner

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/objectserver"
	"os"
	"path"
	"syscall"
)

func loadImageDataBase(baseDir string, objSrv objectserver.ObjectServer) (
	*ImageDataBase, error) {
	fi, err := os.Stat(baseDir)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Cannot stat: %s\t%s\n", baseDir, err))
	}
	if !fi.IsDir() {
		return nil, errors.New(fmt.Sprintf("%s is not a directory\n", baseDir))
	}
	imdb := new(ImageDataBase)
	imdb.baseDir = baseDir
	imdb.imageMap = make(map[string]*image.Image)
	imdb.objectServer = objSrv
	if err = imdb.scanDirectory(""); err != nil {
		return nil, err
	}
	return imdb, nil
}

func (imdb *ImageDataBase) scanDirectory(dirname string) error {
	file, err := os.Open(path.Join(imdb.baseDir, dirname))
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	for _, name := range names {
		filename := path.Join(dirname, name)
		var stat syscall.Stat_t
		err := syscall.Lstat(path.Join(imdb.baseDir, filename), &stat)
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return err
		}
		if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
			err = imdb.scanDirectory(filename)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFREG {
			err = imdb.loadFile(filename)
		}
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return err
		}
	}
	return nil
}

func (imdb *ImageDataBase) loadFile(filename string) error {
	file, err := os.Open(path.Join(imdb.baseDir, filename))
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	var image image.Image
	if err = decoder.Decode(&image); err != nil {
		return err
	}
	image.FileSystem.RebuildInodePointers()
	imdb.imageMap[filename] = &image
	return nil
}
