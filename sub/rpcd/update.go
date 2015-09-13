package rpcd

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/proto/sub"
	"path"
	"time"
)

func (t *rpcType) Update(request sub.UpdateRequest,
	reply *sub.UpdateResponse) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	fs := fileSystemHistory.FileSystem()
	if fs == nil {
		return errors.New("No file-system history yet")
	}
	logger.Printf("Update()\n")
	if fetchInProgress {
		logger.Println("Error: fetch already in progress")
		return errors.New("fetch already in progress")
	}
	if updateInProgress {
		logger.Println("Error: update progress")
		return errors.New("update in progress")
	}
	updateInProgress = true
	go doUpdate(request, fs.RootDirectoryName())
	return nil
}

func doUpdate(request sub.UpdateRequest, rootDirectoryName string) {
	defer clearUpdateInProgress()
	processDeletes(request, rootDirectoryName)
	// TODO(rgooch): Remove debugging hack and implement.
	time.Sleep(time.Second * 15)
	logger.Printf("Update() complete\n")
}

func clearUpdateInProgress() {
	rwLock.Lock()
	defer rwLock.Unlock()
	updateInProgress = false
}

func processDeletes(request sub.UpdateRequest, rootDirectoryName string) {
	for _, pathname := range request.PathsToDelete {
		fullPathname := path.Join(rootDirectoryName, pathname)
		fmt.Println(fullPathname) // TODO(rgooch): Remove debugging.
	}
}
