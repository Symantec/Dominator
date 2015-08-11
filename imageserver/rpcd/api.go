package rpcd

import (
	"github.com/Symantec/Dominator/imageserver/scanner"
	"net/rpc"
)

type rpcType int

var imageDataBase *scanner.ImageDataBase

func Setup(imdb *scanner.ImageDataBase) {
	imageDataBase = imdb
	rpc.RegisterName("ImageServer", new(rpcType))
}
