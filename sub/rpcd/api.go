package rpcd

import (
	"github.com/Symantec/Dominator/sub/scanner"
	"net/rpc"
)

var onlyFsh *scanner.FileSystemHistory

type Subd int

func Setup(fsh *scanner.FileSystemHistory) {
	onlyFsh = fsh
	rpc.Register(new(Subd))
	rpc.HandleHTTP()
}
