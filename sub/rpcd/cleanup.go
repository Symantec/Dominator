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
	t.disableScannerFunc(true)
	defer t.disableScannerFunc(false)
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	t.logger.Printf("Cleanup(): %d objects\n", len(request.Hashes))
	if t.fetchInProgress {
		t.logger.Println("Error: fetch in progress")
		return errors.New("fetch in progress")
	}
	if t.updateInProgress {
		t.logger.Println("Error: update progress")
		return errors.New("update in progress")
	}
	for _, hash := range request.Hashes {
		pathname := path.Join(t.objectsDir, objectcache.HashToFilename(hash))
		err := os.Remove(pathname)
		if err == nil {
			t.logger.Printf("Deleted: %s\n", pathname)
		} else {
			t.logger.Println(err)
		}
	}
	return nil
}
