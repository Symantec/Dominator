package rpcd

import (
	"github.com/Symantec/Dominator/lib/rateio"
	"github.com/Symantec/Dominator/sub/scanner"
	"net/rpc"
	"sync"
)

var fileSystemHistory *scanner.FileSystemHistory
var objectsDir string
var networkReaderContext *rateio.ReaderContext
var rescanObjectCacheChannel chan bool

type rpcType int

func Setup(fsh *scanner.FileSystemHistory, objectsDirname string,
	netReaderContext *rateio.ReaderContext) chan bool {
	fileSystemHistory = fsh
	objectsDir = objectsDirname
	networkReaderContext = netReaderContext
	rescanObjectCacheChannel = make(chan bool)
	rpc.RegisterName("Subd", new(rpcType))
	rpc.HandleHTTP()
	return rescanObjectCacheChannel
}

var (
	rwLock               sync.RWMutex
	fetchInProgress      bool // Fetch() and Update() are mutually exclusive.
	updateInProgress     bool
	startTimeNanoSeconds int32 // For Fetch() or Update().
	startTimeSeconds     int64
)
