package rpcd

import (
	"github.com/Symantec/Dominator/imageserver/scanner"
	"log"
	"net/rpc"
)

type rpcType struct {
	imageDataBase *scanner.ImageDataBase
	logger        *log.Logger
}

func Setup(imdb *scanner.ImageDataBase, lg *log.Logger) {
	rpcObj := &rpcType{
		imageDataBase: imdb,
		logger:        lg}
	rpc.RegisterName("ImageServer", rpcObj)
}
