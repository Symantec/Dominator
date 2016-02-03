package rpcd

import (
	"github.com/Symantec/Dominator/lib/rateio"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/sub/scanner"
	"log"
	"sync"
)

type rpcType struct {
	scannerConfiguration         *scanner.Configuration
	fileSystemHistory            *scanner.FileSystemHistory
	objectsDir                   string
	rootDir                      string
	networkReaderContext         *rateio.ReaderContext
	netbenchFilename             string
	oldTriggersFilename          string
	rescanObjectCacheChannel     chan<- bool
	disableScannerFunc           func(disableScanner bool)
	logger                       *log.Logger
	rwLock                       sync.RWMutex
	pollLock                     sync.Mutex
	getFilesLock                 sync.Mutex
	fetchInProgress              bool // Fetch() & Update() mutually exclusive.
	updateInProgress             bool
	startTimeNanoSeconds         int32 // For Fetch() or Update().
	startTimeSeconds             int64
	lastFetchError               error
	lastUpdateError              error
	lastUpdateHadTriggerFailures bool
}

func Setup(configuration *scanner.Configuration, fsh *scanner.FileSystemHistory,
	objectsDirname string, rootDirname string,
	netReaderContext *rateio.ReaderContext,
	netbenchFname string, oldTriggersFname string,
	disableScannerFunction func(disableScanner bool),
	logger *log.Logger) <-chan bool {
	rescanObjectCacheChannel := make(chan bool)
	rpcObj := &rpcType{
		scannerConfiguration:     configuration,
		fileSystemHistory:        fsh,
		objectsDir:               objectsDirname,
		rootDir:                  rootDirname,
		networkReaderContext:     netReaderContext,
		netbenchFilename:         netbenchFname,
		oldTriggersFilename:      oldTriggersFname,
		rescanObjectCacheChannel: rescanObjectCacheChannel,
		disableScannerFunc:       disableScannerFunction,
		logger:                   logger}
	srpc.RegisterName("Subd", rpcObj)
	return rescanObjectCacheChannel
}
