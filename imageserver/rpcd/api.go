package rpcd

import (
	"github.com/Symantec/Dominator/imageserver/scanner"
	"github.com/Symantec/Dominator/objectserver"
	"net/rpc"
)

type ImageServer int

var imageDataBase *scanner.ImageDataBase
var objectServer objectserver.ObjectServer

func Setup(imdb *scanner.ImageDataBase, objSrv objectserver.ObjectServer) {
	imageDataBase = imdb
	objectServer = objSrv
	rpc.Register(new(ImageServer))
	rpc.HandleHTTP()
}
