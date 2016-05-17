package scanner

import (
	"log"
	"runtime"
	"syscall"
)

var disableScanRequest chan bool
var disableScanAcknowledge chan bool

func startScannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration, logger *log.Logger) (
	<-chan *FileSystem, func(disableScanner bool)) {
	fsChannel := make(chan *FileSystem)
	disableScanRequest = make(chan bool, 1)
	disableScanAcknowledge = make(chan bool)
	go scannerDaemon(rootDirectoryName, cacheDirectoryName, configuration,
		fsChannel, logger)
	return fsChannel, doDisableScanner
}

func startScanning(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration, logger *log.Logger,
	mainFunc func(<-chan *FileSystem, func(disableScanner bool))) {
	fsChannel := make(chan *FileSystem)
	disableScanRequest = make(chan bool, 1)
	disableScanAcknowledge = make(chan bool)
	go mainFunc(fsChannel, doDisableScanner)
	scannerDaemon(rootDirectoryName, cacheDirectoryName, configuration,
		fsChannel, logger)
}

func scannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration, fsChannel chan<- *FileSystem,
	logger *log.Logger) {
	runtime.LockOSThread()
	loweredPriority := false
	var oldFS FileSystem
	for {
		fs, err := scanFileSystem(rootDirectoryName, cacheDirectoryName,
			configuration, &oldFS)
		if err != nil {
			if err.Error() == "DisableScan" {
				disableScanAcknowledge <- true
				<-disableScanAcknowledge
				continue
			}
			logger.Printf("Error scanning\t%s\n", err)
		} else {
			oldFS.InodeTable = fs.InodeTable
			oldFS.DirectoryInode = fs.DirectoryInode
			fsChannel <- fs
			runtime.GC()
			if !loweredPriority {
				syscall.Setpriority(syscall.PRIO_PROCESS, 0, 10)
				loweredPriority = true
			}
		}
	}
}

func doDisableScanner(disableScanner bool) {
	if disableScanner {
		disableScanRequest <- true
		<-disableScanAcknowledge
	} else {
		disableScanAcknowledge <- true
	}
}

func checkScanDisableRequest() bool {
	if len(disableScanRequest) > 0 {
		<-disableScanRequest
		return true
	}
	return false
}
