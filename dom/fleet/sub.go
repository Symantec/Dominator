package fleet

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/scanner"
	"net/rpc"
	"strings"
)

func (fleet *Fleet) scanNextSub() {
	if fleet.nextSubToScan >= uint(len(fleet.subs)) {
		fleet.nextSubToScan = 0
		return
	}
	sub := fleet.subs[fleet.nextSubToScan]
	fleet.nextSubToScan++
	if sub.connection == nil {
		hostname := strings.SplitN(sub.hostname, "*", 2)[0]
		var err error
		sub.connection, err = rpc.DialHTTP("tcp", hostname+":6969")
		if err != nil {
			fmt.Printf("Error dialing\t%s\n", err)
			return
		}
	}
	var reply *scanner.FileSystem
	err := sub.connection.Call("Subd.Poll", sub.generationCount, &reply)
	if err != nil {
		fmt.Printf("Error calling\t%s\n", err)
		return
	}
	reply.RebuildPointers()
	//sub.generationCount = reply.GenerationCount()
	fmt.Printf("Scanned: %s\n", sub.hostname)
}
