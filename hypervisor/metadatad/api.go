package metadatad

import (
	"io"
	"net"
	"net/http"

	"github.com/Symantec/Dominator/hypervisor/manager"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type rawHandlerFunc func(w http.ResponseWriter, ipAddr net.IP)
type metadataWriter func(writer io.Writer, vmInfo proto.VmInfo) error

type server struct {
	bridges      []net.Interface
	manager      *manager.Manager
	logger       log.DebugLogger
	infoHandlers map[string]metadataWriter
	rawHandlers  map[string]rawHandlerFunc
}

func StartServer(bridges []net.Interface, managerObj *manager.Manager,
	logger log.DebugLogger) error {
	s := &server{
		bridges: bridges,
		manager: managerObj,
		logger:  logger,
	}
	s.infoHandlers = map[string]metadataWriter{
		"/latest/dynamic/epoch-time":                 s.showTime,
		"/latest/dynamic/instance-identity/document": s.showVM,
	}
	s.rawHandlers = map[string]rawHandlerFunc{
		"/latest/user-data": s.showUserData,
	}
	return s.startServer()
}
