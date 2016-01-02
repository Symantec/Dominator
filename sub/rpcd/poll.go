package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) Poll(conn *srpc.Conn) {
	defer conn.Flush()
	t.pollLock.Lock()
	defer t.pollLock.Unlock()
	var request sub.PollRequest
	var response sub.PollResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		conn.WriteString(err.Error() + "\n")
		return
	}
	conn.WriteString("\n")
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
		response.FileSystem = &fs.FileSystem
		response.ObjectCache = fs.ObjectCache
	}
	encoder := gob.NewEncoder(conn)
	encoder.Encode(response)
}
