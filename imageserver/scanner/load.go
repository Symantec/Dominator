package scanner

import (
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/objectserver"
)

func loadImageDataBase(baseDir string, objSrv objectserver.ObjectServer) (
	*ImageDataBase, error) {
	imdb := new(ImageDataBase)
	imdb.imageMap = make(map[string]*image.Image)
	imdb.objectServer = objSrv
	return imdb, nil
}
