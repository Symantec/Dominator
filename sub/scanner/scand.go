package scanner

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"runtime"
	"syscall"
)

func startScannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext) chan *FileSystem {
	fsChannel := make(chan *FileSystem)
	go scannerDaemon(rootDirectoryName, cacheDirectoryName, ctx, fsChannel)
	return fsChannel
}

func scannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext, fsChannel chan *FileSystem) {
	if runtime.GOMAXPROCS(0) < 2 {
		runtime.GOMAXPROCS(2)
	}
	runtime.LockOSThread()
	loweredPriority := false
	for {
		fs, err := ScanFileSystem(rootDirectoryName, cacheDirectoryName, ctx)
		if err != nil {
			fmt.Printf("Error scanning\t%s\n", err)
		} else {
			fsChannel <- fs
			if !loweredPriority {
				syscall.Setpriority(syscall.PRIO_PROCESS, 0, 10)
				loweredPriority = true
			}
		}
	}
}
