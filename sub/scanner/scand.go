package scanner

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/fsrateio"
)

func startScannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext) chan *FileSystem {
	fsChannel := make(chan *FileSystem)
	go scannerDaemon(rootDirectoryName, cacheDirectoryName, ctx, fsChannel)
	return fsChannel
}

func scannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext, fsChannel chan *FileSystem) {
	for {
		fs, err := ScanFileSystem(rootDirectoryName, cacheDirectoryName, ctx)
		if err != nil {
			fmt.Printf("Error scanning\t%s\n", err)
		} else {
			fsChannel <- fs
		}
	}
}
