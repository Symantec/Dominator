package rpcd

import (
	"io"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/tricorder/go/tricorder"
	"github.com/Symantec/tricorder/go/tricorder/units"
)

type srpcType struct {
	objectServer      objectserver.StashingObjectServer
	replicationMaster string
	getSemaphore      chan bool
	logger            log.DebugLogger
}

type htmlWriter struct {
	getSemaphore chan bool
}

func (hw *htmlWriter) WriteHtml(writer io.Writer) {
	hw.writeHtml(writer)
}

func Setup(objSrv objectserver.StashingObjectServer, replicationMaster string,
	logger log.DebugLogger) *htmlWriter {
	getSemaphore := make(chan bool, 100)
	srpcObj := &srpcType{objSrv, replicationMaster, getSemaphore, logger}
	srpc.RegisterName("ObjectServer", srpcObj)
	tricorder.RegisterMetric("/get-requests",
		func() uint { return uint(len(getSemaphore)) },
		units.None, "number of GetObjects() requests in progress")
	return &htmlWriter{getSemaphore}
}
