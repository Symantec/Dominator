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
)

func (t *rpcType) Fetch(request sub.FetchRequest,
	reply *sub.FetchResponse) error {
	if *readOnly {
		txt := "Fetch() rejected due to read-only mode"
		t.logger.Println(txt)
		return errors.New(txt)
	}
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	t.logger.Printf("Fetch() %d objects\n", len(request.Hashes))
	if t.fetchInProgress {
		t.logger.Println("Error: fetch already in progress")
		return errors.New("fetch already in progress")
	}
	if t.updateInProgress {
		t.logger.Println("Error: update in progress")
		return errors.New("update in progress")
	}
	t.fetchInProgress = true
	go t.doFetch(request)
	return nil
}

func (t *rpcType) doFetch(request sub.FetchRequest) {
	defer t.clearFetchInProgress()
	objectServer := objectclient.NewObjectClient(request.ServerAddress)
	benchmark := false
	if t.networkReaderContext.MaximumSpeed() < 1 {
		benchmark = enoughBytesForBenchmark(objectServer, request)
		if benchmark {
			objectServer.SetExclusiveGetObjects(true)
			t.logger.Println("Benchmarking network speed")
		}
	}
	objectsReader, err := objectServer.GetObjects(request.Hashes)
	if err != nil {
		t.logger.Printf("Error getting object reader:\t%s\n", err.Error())
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
			t.logger.Println(err)
			if *exitOnFetchFailure {
				os.Exit(1)
			}
			return
		}
		err = readOne(t.objectsDir, hash, length,
			t.networkReaderContext.NewReader(reader))
		reader.Close()
		if err != nil {
			t.logger.Println(err)
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
		file, err := os.Create(t.netbenchFilename)
		if err == nil {
			fmt.Fprintf(file, "%d\n", speed)
			file.Close()
		}
		t.networkReaderContext.InitialiseMaximumSpeed(speed)
	}
	t.logger.Printf("Fetch() complete. Read: %s in %s (%s/s)\n",
		format.FormatBytes(totalLength), duration, format.FormatBytes(speed))
	t.rescanObjectCacheChannel <- true
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

func readOne(objectsDir string, hash hash.Hash, length uint64,
	reader io.Reader) error {
	filename := path.Join(objectsDir, objectcache.HashToFilename(hash))
	dirname := path.Dir(filename)
	if err := os.MkdirAll(dirname, syscall.S_IRWXU); err != nil {
		return err
	}
	return fsutil.CopyToFile(filename, reader, int64(length))
}

func (t *rpcType) clearFetchInProgress() {
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	t.fetchInProgress = false
}
