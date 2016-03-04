package lib

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/objectserver"
	"log"
)

func AddObjects(conn *srpc.Conn, objSrv objectserver.ObjectServer,
	logger *log.Logger) error {
	return addObjects(conn, objSrv, logger)
}
