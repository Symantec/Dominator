package rpcd

import (
	"github.com/Symantec/Dominator/lib/rateio"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/sub/scanner"
	"log"
	"net/rpc"
	"sync"
)

type rpcType struct {
	scannerConfiguration         *scanner.Configuration
	fileSystemHistory            *scanner.FileSystemHistory
	objectsDir                   string
	networkReaderContext         *rateio.ReaderContext
	netbenchFilename             string
	oldTriggersFilename          string
	rescanObjectCacheChannel     chan<- bool
	disableScannerFunc           func(disableScanner bool)
	logger                       *log.Logger
	rwLock                       sync.RWMutex
	fetchInProgress              bool // Fetch() & Update() mutually exclusive.
	updateInProgress             bool
	startTimeNanoSeconds         int32 // For Fetch() or Update().
	startTimeSeconds             int64
	lastUpdateHadTriggerFailures bool
}

func Setup(configuration *scanner.Configuration, fsh *scanner.FileSystemHistory,
	objectsDirname string, netReaderContext *rateio.ReaderContext,
	netbenchFname string, oldTriggersFname string,
	disableScannerFunction func(disableScanner bool),
	logger *log.Logger) <-chan bool {
	rescanObjectCacheChannel := make(chan bool)
	rpcObj := &rpcType{
		scannerConfiguration:     configuration,
		fileSystemHistory:        fsh,
		objectsDir:               objectsDirname,
		networkReaderContext:     netReaderContext,
		netbenchFilename:         netbenchFname,
		oldTriggersFilename:      oldTriggersFname,
		rescanObjectCacheChannel: rescanObjectCacheChannel,
		disableScannerFunc:       disableScannerFunction,
		logger:                   logger}
	rpc.RegisterName("Subd", rpcObj)
	srpc.RegisterName("Subd", rpcObj)
	rpc.HandleHTTP()
	return rescanObjectCacheChannel
}
