package herd

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"net/rpc"
	"strings"
)

func (herd *Herd) pollNextSub() bool {
	if herd.nextSubToPoll >= uint(len(herd.subsByIndex)) {
		herd.nextSubToPoll = 0
		return true
	}
	sub := herd.subsByIndex[herd.nextSubToPoll]
	herd.nextSubToPoll++
	if sub.connection == nil {
		hostname := strings.SplitN(sub.hostname, "*", 2)[0]
		var err error
		sub.connection, err = rpc.DialHTTP("tcp",
			fmt.Sprintf("%s:%d", hostname, constants.SubPortNumber))
		if err != nil {
			fmt.Printf("Error dialing\t%s\n", err)
			return false
		}
	}
	var request subproto.PollRequest
	request.HaveGeneration = sub.generationCount
	var reply subproto.PollResponse
	err := sub.connection.Call("Subd.Poll", request, &reply)
	if err != nil {
		fmt.Printf("Error calling\t%s\n", err)
		return false
	}
	fs := reply.FileSystem
	if fs != nil {
		fs.RebuildPointers()
		sub.fileSystem = fs
		sub.generationCount = reply.GenerationCount
		fmt.Printf("Polled: %s, GenerationCount=%d\n",
			sub.hostname, reply.GenerationCount)
	}
	return false
}
