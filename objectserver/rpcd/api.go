package rpcd

import (
	"github.com/Symantec/Dominator/objectserver"
	"log"
	"net/http"
	"net/rpc"
)

type rpcType int

var objectServer objectserver.ObjectServer
var logger *log.Logger

func Setup(objSrv objectserver.ObjectServer, lg *log.Logger) {
	objectServer = objSrv
	logger = lg
	rpc.RegisterName("ObjectServer", new(rpcType))
	http.HandleFunc("/GetObjects", getObjectsHandler)
}
