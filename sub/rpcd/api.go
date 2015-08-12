package rpcd

import (
	"github.com/Symantec/Dominator/sub/scanner"
	"net/rpc"
)

var fileSystemHistory *scanner.FileSystemHistory

type rpcType int

func Setup(fsh *scanner.FileSystemHistory) {
	fileSystemHistory = fsh
	rpc.RegisterName("Subd", new(rpcType))
	rpc.HandleHTTP()
}
