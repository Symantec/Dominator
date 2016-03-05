package rpcd

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"io"
	"os"
	"path"
	"syscall"
	"time"
)

const filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP

var (
	exitOnFetchFailure = flag.Bool("exitOnFetchFailure", false,
		"If true, exit if there are fetch failures. For debugging only")
)

func (t *rpcType) Fetch(conn *srpc.Conn) error {
	defer conn.Flush()
	var request sub.FetchRequest
	var response sub.FetchResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if err := t.fetch(request, &response); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response)
}

func (t *rpcType) fetch(request sub.FetchRequest,
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
	go func() {
		t.lastFetchError = t.doFetch(request)
		if t.lastFetchError != nil && *exitOnFetchFailure {
			os.Exit(1)
		}
	}()
	return nil
}

func (t *rpcType) doFetch(request sub.FetchRequest) error {
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
		return err
	}
	defer objectsReader.Close()
	var totalLength uint64
	defer func() { t.rescanObjectCacheChannel <- true }()
	timeStart := time.Now()
	for _, hash := range request.Hashes {
		length, reader, err := objectsReader.NextObject()
		if err != nil {
			t.logger.Println(err)
			return err
		}
		err = readOne(t.objectsDir, hash, length,
			t.networkReaderContext.NewReader(reader))
		reader.Close()
		if err != nil {
			t.logger.Println(err)
			return err
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
	return nil
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
	return fsutil.CopyToFile(filename, filePerms, reader, length)
}

func (t *rpcType) clearFetchInProgress() {
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	t.fetchInProgress = false
}
