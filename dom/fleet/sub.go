package fleet

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/httpd"
	"net/rpc"
	"strings"
)

func (fleet *Fleet) pollNextSub() {
	if fleet.nextSubToPoll >= uint(len(fleet.subs)) {
		fleet.nextSubToPoll = 0
		return
	}
	sub := fleet.subs[fleet.nextSubToPoll]
	fleet.nextSubToPoll++
	if sub.connection == nil {
		hostname := strings.SplitN(sub.hostname, "*", 2)[0]
		var err error
		sub.connection, err = rpc.DialHTTP("tcp", hostname+":6969")
		if err != nil {
			fmt.Printf("Error dialing\t%s\n", err)
			return
		}
	}
	var reply *httpd.FileSystemHistory
	err := sub.connection.Call("Subd.Poll", sub.generationCount, &reply)
	if err != nil {
		fmt.Printf("Error calling\t%s\n", err)
		return
	}
	fs := reply.FileSystem
	if fs != nil {
		fs.RebuildPointers()
		sub.generationCount = reply.GenerationCount
		fmt.Printf("Polled: %s, GenerationCount=%d\n",
			sub.hostname, reply.GenerationCount)
	}
}
