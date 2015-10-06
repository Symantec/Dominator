package scanner

import (
	"log"
	"runtime"
	"syscall"
)

func startScannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration, logger *log.Logger) chan *FileSystem {
	fsChannel := make(chan *FileSystem)
	go scannerDaemon(rootDirectoryName, cacheDirectoryName, configuration,
		fsChannel, logger)
	return fsChannel
}

func scannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration, fsChannel chan *FileSystem,
	logger *log.Logger) {
	runtime.LockOSThread()
	loweredPriority := false
	var oldFS FileSystem
	for {
		fs, err := scanFileSystem(rootDirectoryName, cacheDirectoryName,
			configuration, &oldFS)
		if err != nil {
			logger.Printf("Error scanning\t%s\n", err)
		} else {
			oldFS.InodeTable = fs.InodeTable
			fsChannel <- fs
			runtime.GC()
			if !loweredPriority {
				syscall.Setpriority(syscall.PRIO_PROCESS, 0, 10)
				loweredPriority = true
			}
		}
	}
}
