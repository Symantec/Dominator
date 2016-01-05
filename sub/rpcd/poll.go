package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) Poll(conn *srpc.Conn) error {
	defer conn.Flush()
	t.pollLock.Lock()
	defer t.pollLock.Unlock()
	var request sub.PollRequest
	var response sub.PollResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	response.NetworkSpeed = t.networkReaderContext.MaximumSpeed()
	t.rwLock.RLock()
	response.FetchInProgress = t.fetchInProgress
	response.UpdateInProgress = t.updateInProgress
	response.LastUpdateHadTriggerFailures = t.lastUpdateHadTriggerFailures
	t.rwLock.RUnlock()
	response.GenerationCount = t.fileSystemHistory.GenerationCount()
	fs := t.fileSystemHistory.FileSystem()
	if fs != nil &&
		request.HaveGeneration != t.fileSystemHistory.GenerationCount() {
		response.FileSystemFollows = true
	}
	encoder := gob.NewEncoder(conn)
	if err := encoder.Encode(response); err != nil {
		return err
	}
	if response.FileSystemFollows {
		if err := fs.FileSystem.Encode(conn); err != nil {
			return err
		}
		if err := fs.ObjectCache.Encode(conn); err != nil {
			return err
		}
	}
	return nil
}
