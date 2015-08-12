package scanner

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/objectserver"
	"os"
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
	return imdb, nil
}
