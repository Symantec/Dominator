package rpcd

import (
	"github.com/Symantec/Dominator/sub/scanner"
	"net/rpc"
)

var onlyFsh *scanner.FileSystemHistory

type Subd int

func Setup(fsh *scanner.FileSystemHistory) {
	onlyFsh = fsh
	subd := new(Subd)
	rpc.Register(subd)
	rpc.HandleHTTP()
}
