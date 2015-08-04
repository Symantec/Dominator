package rpcd

import (
	"github.com/Symantec/Dominator/imageserver/scanner"
	"net/rpc"
)

type ImageServer int

var imageDataBase *scanner.ImageDataBase

func Setup(imdb *scanner.ImageDataBase) {
	imageDataBase = imdb
	rpc.Register(new(ImageServer))
	rpc.HandleHTTP()
}
