package rpcd

import (
	"github.com/Symantec/Dominator/imageserver/scanner"
	"github.com/Symantec/Dominator/lib/srpc"
	"log"
	"net/rpc"
)

type rpcType struct {
	imageDataBase *scanner.ImageDataBase
	logger        *log.Logger
}

type srpcType struct {
	imageDataBase *scanner.ImageDataBase
	logger        *log.Logger
}

func Setup(imdb *scanner.ImageDataBase, lg *log.Logger) {
	rpcObj := rpcType{
		imageDataBase: imdb,
		logger:        lg}
	rpc.RegisterName("ImageServer", &rpcObj)
	srpcObj := srpcType(rpcObj)
	srpc.RegisterName("ImageServer", &srpcObj)
}
