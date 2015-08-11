package rpcd

import (
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) Poll(request sub.PollRequest, reply *sub.PollResponse) error {
	fs := fileSystemHistory.FileSystem()
	if fs != nil &&
		request.HaveGeneration != fileSystemHistory.GenerationCount() {
		var response sub.PollResponse
		response.GenerationCount = fileSystemHistory.GenerationCount()
		response.FileSystem = fs
		*reply = response
	}
	return nil
}
