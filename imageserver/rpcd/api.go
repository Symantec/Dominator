package rpcd

import (
	"github.com/Symantec/Dominator/imageserver/scanner"
	"log"
	"net/rpc"
)

type rpcType int

var imageDataBase *scanner.ImageDataBase
var logger *log.Logger

func Setup(imdb *scanner.ImageDataBase, lg *log.Logger) {
	imageDataBase = imdb
	logger = lg
	rpc.RegisterName("ImageServer", new(rpcType))
}
