package filesystem

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"os"
)

func newObjectServer(baseDir string) (*FileSystemObjectServer, error) {
	fi, err := os.Stat(baseDir)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Cannot stat: %s\t%s\n", baseDir, err))
	}
	if !fi.IsDir() {
		return nil, errors.New(fmt.Sprintf("%s is not a directory\n", baseDir))
	}
	var objSrv FileSystemObjectServer
	objSrv.baseDir = baseDir
	objSrv.checkMap = make(map[hash.Hash]bool)
	return &objSrv, nil
}
