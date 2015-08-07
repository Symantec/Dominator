package scanner

import (
	"fmt"
	"runtime"
	"syscall"
)

func startScannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration) chan *FileSystem {
	fsChannel := make(chan *FileSystem)
	go scannerDaemon(rootDirectoryName, cacheDirectoryName, configuration,
		fsChannel)
	return fsChannel
}

func scannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration, fsChannel chan *FileSystem) {
	if runtime.GOMAXPROCS(0) < 2 {
		runtime.GOMAXPROCS(2)
	}
	runtime.LockOSThread()
	loweredPriority := false
	var oldFS FileSystem
	for {
		fs, err := scanFileSystem(rootDirectoryName, cacheDirectoryName,
			configuration, &oldFS)
		if err != nil {
			fmt.Printf("Error scanning\t%s\n", err)
		} else {
			oldFS.RegularInodeTable = fs.RegularInodeTable
			oldFS.SymlinkInodeTable = fs.SymlinkInodeTable
			oldFS.InodeTable = fs.InodeTable
			fsChannel <- fs
			if !loweredPriority {
				syscall.Setpriority(syscall.PRIO_PROCESS, 0, 10)
				loweredPriority = true
			}
		}
	}
}
