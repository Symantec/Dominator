package rpcd

import (
	"github.com/Symantec/Dominator/sub/scanner"
	"net/rpc"
	"sync"
)

var fileSystemHistory *scanner.FileSystemHistory
var objectsDir string

type rpcType int

func Setup(fsh *scanner.FileSystemHistory, objectsDirname string) {
	fileSystemHistory = fsh
	objectsDir = objectsDirname
	rpc.RegisterName("Subd", new(rpcType))
	rpc.HandleHTTP()
}

var (
	rwLock               sync.RWMutex
	fetchInProgress      bool // Fetch() and Update() are mutually exclusive.
	updateInProgress     bool
	startTimeNanoSeconds int32 // For Fetch() or Update().
	startTimeSeconds     int64
)
