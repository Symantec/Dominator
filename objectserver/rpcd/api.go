package rpcd

import (
	"github.com/Symantec/Dominator/objectserver"
	"io"
	"log"
	"net/http"
	"net/rpc"
)

type rpcType int

var objectServer objectserver.ObjectServer
var logger *log.Logger
var getSemaphore chan bool = make(chan bool, 100)

type htmlWriter struct{}

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

func Setup(objSrv objectserver.ObjectServer, lg *log.Logger) *htmlWriter {
	objectServer = objSrv
	logger = lg
	rpc.RegisterName("ObjectServer", new(rpcType))
	http.HandleFunc("/GetObjects", getObjectsHandler)
	return &htmlWriter{}
}
