package filesystem

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"os"
)

func newObjectServer(baseDir string) (*ObjectServer, error) {
	fi, err := os.Stat(baseDir)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Cannot stat: %s\t%s\n", baseDir, err))
	}
	if !fi.IsDir() {
		return nil, errors.New(fmt.Sprintf("%s is not a directory\n", baseDir))
	}
	var objSrv ObjectServer
	objSrv.baseDir = baseDir
	objSrv.checkMap = make(map[hash.Hash]bool)
	cache, err := objectcache.ScanObjectCache(baseDir)
	if err != nil {
		return nil, err
	}
	for _, hash := range cache {
		objSrv.checkMap[hash] = true
	}
	return &objSrv, nil
}
