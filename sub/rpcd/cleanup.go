package rpcd

import (
	"errors"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/proto/sub"
	"os"
	"path"
)

func (t *rpcType) Cleanup(request sub.CleanupRequest,
	reply *sub.CleanupResponse) error {
	disableScannerFunc(true)
	defer disableScannerFunc(false)
	rwLock.Lock()
	defer rwLock.Unlock()
	logger.Printf("Cleanup(): %d objects\n", len(request.Hashes))
	if fetchInProgress {
		logger.Println("Error: fetch in progress")
		return errors.New("fetch in progress")
	}
	if updateInProgress {
		logger.Println("Error: update progress")
		return errors.New("update in progress")
	}
	for _, hash := range request.Hashes {
		pathname := path.Join(objectsDir, objectcache.HashToFilename(hash))
		err := os.Remove(pathname)
		if err == nil {
			logger.Printf("Deleted: %s\n", pathname)
		} else {
			logger.Println(err)
		}
	}
	return nil
}
