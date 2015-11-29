package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/objectserver"
	"io"
	"log"
	"net/rpc"
)

type objectServer struct {
	objectServer objectserver.ObjectServer
}

type srpcType struct {
	objectServer objectserver.ObjectServer
}

var logger *log.Logger
var getSemaphore chan bool = make(chan bool, 100)

type htmlWriter struct{}

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

func Setup(objSrv objectserver.ObjectServer, lg *log.Logger) *htmlWriter {
	rpcObj := &objectServer{objSrv}
	srpcObj := &srpcType{objSrv}
	logger = lg
	rpc.RegisterName("ObjectServer", rpcObj)
	srpc.RegisterName("ObjectServer", srpcObj)
	return &htmlWriter{}
}
