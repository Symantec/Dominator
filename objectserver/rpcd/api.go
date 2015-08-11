package rpcd

import (
	"github.com/Symantec/Dominator/objectserver"
	"net/rpc"
)

type rpcType int

var objectServer objectserver.ObjectServer

func Setup(objSrv objectserver.ObjectServer) {
	objectServer = objSrv
	rpc.RegisterName("ObjectServer", new(rpcType))
	rpc.HandleHTTP()
}
