package rpcd

import (
	"errors"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/proto/sub"
	"io"
	"os"
	"path"
	"syscall"
	"time"
)

var (
	exitOnFetchFailure = flag.Bool("exitOnFetchFailure", false,
		"If true, exit if there are fetch failures. For debugging only")
	// TODO(rgooch): Remove this flag once data corruption is fixed, so that
	//               scanning always continues during a fetch.
	stopScanDuringFetch = flag.Bool("stopScanDuringFetch", true,
		"If true, stop scan during fetching. This reduces the chance of fetch problems")
)

func (t *rpcType) Fetch(request sub.FetchRequest,
	reply *sub.FetchResponse) error {
	if *readOnly {
		txt := "Fetch() rejected due to read-only mode"
		logger.Println(txt)
		return errors.New(txt)
	}
	rwLock.Lock()
	defer rwLock.Unlock()
	logger.Printf("Fetch() %d objects\n", len(request.Hashes))
	if fetchInProgress {
		logger.Println("Error: fetch already in progress")
		return errors.New("fetch already in progress")
	}
	if updateInProgress {
		logger.Println("Error: update in progress")
		return errors.New("update in progress")
	}
	fetchInProgress = true
	go doFetch(request)
	return nil
}

func doFetch(request sub.FetchRequest) {
	defer clearFetchInProgress()
	if *stopScanDuringFetch {
		disableScannerFunc(true)
		defer disableScannerFunc(false)
	}
	objectServer := objectclient.NewObjectClient(request.ServerAddress)
	benchmark := false
	if networkReaderContext.MaximumSpeed() < 1 {
		benchmark = enoughBytesForBenchmark(objectServer, request)
		if benchmark {
			objectServer.SetExclusiveGetObjects(true)
			logger.Println("Benchmarking network speed")
		}
	}
	objectsReader, err := objectServer.GetObjects(request.Hashes)
	if err != nil {
		logger.Printf("Error getting object reader:\t%s\n", err.Error())
		if *exitOnFetchFailure {
			os.Exit(1)
		}
		return
	}
	defer objectsReader.Close()
	var totalLength uint64
	timeStart := time.Now()
	for _, hash := range request.Hashes {
		length, reader, err := objectsReader.NextObject()
		if err != nil {
			logger.Println(err)
			if *exitOnFetchFailure {
				os.Exit(1)
			}
			return
		}
		err = readOne(hash, length, networkReaderContext.NewReader(reader))
		reader.Close()
		if err != nil {
			logger.Println(err)
			if *exitOnFetchFailure {
				os.Exit(1)
			}
			return
		}
		totalLength += length
	}
	duration := time.Since(timeStart)
	speed := uint64(float64(totalLength) / duration.Seconds())
	if benchmark {
		file, err := os.Create(netbenchFilename)
		if err == nil {
			fmt.Fprintf(file, "%d\n", speed)
			file.Close()
		}
		networkReaderContext.InitialiseMaximumSpeed(speed)
	}
	logger.Printf("Fetch() complete. Read: %s in %s (%s/s)\n",
		format.FormatBytes(totalLength), duration, format.FormatBytes(speed))
	rescanObjectCacheChannel <- true
}

func enoughBytesForBenchmark(objectServer *objectclient.ObjectClient,
	request sub.FetchRequest) bool {
	lengths, err := objectServer.CheckObjects(request.Hashes)
	if err != nil {
		return false
	}
	var totalLength uint64
	for _, length := range lengths {
		totalLength += length
	}
	if totalLength > 1024*1024*64 {
		return true
	}
	return false
}

func readOne(hash hash.Hash, length uint64, reader io.Reader) error {
	filename := path.Join(objectsDir, objectcache.HashToFilename(hash))
	dirname := path.Dir(filename)
	if err := os.MkdirAll(dirname, syscall.S_IRWXU); err != nil {
		return err
	}
	return fsutil.CopyToFile(filename, reader, int64(length))
}

func clearFetchInProgress() {
	rwLock.Lock()
	defer rwLock.Unlock()
	fetchInProgress = false
}
