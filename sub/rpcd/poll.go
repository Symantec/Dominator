package rpcd

import (
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) Poll(request sub.PollRequest, reply *sub.PollResponse) error {
	var response sub.PollResponse
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
		response.FileSystem = fs
	}
	*reply = response
	return nil
}
