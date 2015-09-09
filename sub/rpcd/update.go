package rpcd

import (
	"errors"
	"github.com/Symantec/Dominator/proto/sub"
	"time"
)

func (t *rpcType) Update(request sub.UpdateRequest,
	reply *sub.UpdateResponse) error {
	rwLock.Lock()
	defer rwLock.Unlock()
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
	go doUpdate(request)
	return nil
}

func doUpdate(request sub.UpdateRequest) {
	defer clearUpdateInProgress()
	// TODO(rgooch): Remove debugging hack and implement.
	time.Sleep(time.Second * 15)
	logger.Printf("Update() complete\n")
}

func clearUpdateInProgress() {
	rwLock.Lock()
	defer rwLock.Unlock()
	updateInProgress = false
}
