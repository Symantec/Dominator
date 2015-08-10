package scanner

import (
	"github.com/Symantec/Dominator/objectserver"
)

func loadImageDataBase(baseDir string, objSrv objectserver.ObjectServer) (
	*ImageDataBase, error) {
	imdb := new(ImageDataBase)
	imdb.objectServer = objSrv
	return imdb, nil
}
