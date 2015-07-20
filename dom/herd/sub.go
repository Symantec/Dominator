package herd

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/httpd"
	"net/rpc"
	"strings"
)

func (herd *Herd) pollNextSub() {
	if herd.nextSubToPoll >= uint(len(herd.subsByIndex)) {
		herd.nextSubToPoll = 0
		return
	}
	sub := herd.subsByIndex[herd.nextSubToPoll]
	herd.nextSubToPoll++
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
