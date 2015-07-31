package rpcd

import (
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *Subd) Poll(request sub.PollRequest, reply *sub.PollResponse) error {
	fs := onlyFsh.FileSystem()
	if fs != nil && request.HaveGeneration != onlyFsh.GenerationCount() {
		var response sub.PollResponse
		response.GenerationCount = onlyFsh.GenerationCount()
		response.FileSystem = fs
		*reply = response
	}
	return nil
}
