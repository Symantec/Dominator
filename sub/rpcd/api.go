package rpcd

import (
	"github.com/Symantec/Dominator/sub/scanner"
	"net/rpc"
)

var fileSystemHistory *scanner.FileSystemHistory

type Subd int

func Setup(fsh *scanner.FileSystemHistory) {
	fileSystemHistory = fsh
	rpc.Register(new(Subd))
	rpc.HandleHTTP()
}
