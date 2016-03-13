package objectserver

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/objectserver/filesystem"
	"log"
	"os"
	"syscall"
)

const dirPerms = syscall.S_IRWXU

func newObjectServer(objDir string, logger *log.Logger) (*ObjectServer, error) {
	fi, err := os.Stat(objDir)
	if err != nil {
		if err := os.Mkdir(objDir, dirPerms); err != nil {
			return nil, err
		}
	} else if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not a directory\n", objDir)
	}
	objectServer := new(ObjectServer)
	objectServer.objSrv, err = filesystem.NewObjectServer(objDir, logger)
	if err != nil {
		return nil, err
	}
	objectServer.logger = logger
	return objectServer, nil
}
