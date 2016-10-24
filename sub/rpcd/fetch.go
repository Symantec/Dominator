package rpcd

import (
	"errors"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/netspeed"
	"github.com/Symantec/Dominator/lib/objectcache"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/rateio"
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

func (t *rpcType) Fetch(conn *srpc.Conn, request sub.FetchRequest,
	reply *sub.FetchResponse) error {
	if *readOnly {
		txt := "Fetch() rejected due to read-only mode"
		t.logger.Println(txt)
		return errors.New(txt)
	}
	if err := t.getFetchLock(); err != nil {
		return err
	}
	if request.Wait {
		return t.fetchAndUnlock(request)
	}
	go t.fetchAndUnlock(request)
	return nil
}

func (t *rpcType) getFetchLock() error {
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	if t.fetchInProgress {
		t.logger.Println("Error: fetch already in progress")
		return errors.New("fetch already in progress")
	}
	if t.updateInProgress {
		t.logger.Println("Error: update in progress")
		return errors.New("update in progress")
	}
	t.fetchInProgress = true
	return nil
}

func (t *rpcType) fetchAndUnlock(request sub.FetchRequest) error {
	err := t.doFetch(request)
	if err != nil && *exitOnFetchFailure {
		os.Exit(1)
	}
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	t.lastFetchError = err
	return err
}

func (t *rpcType) doFetch(request sub.FetchRequest) error {
	defer t.clearFetchInProgress()
	objectServer := objectclient.NewObjectClient(request.ServerAddress)
	defer objectServer.Close()
	benchmark := false
	linkSpeed, haveLinkSpeed := netspeed.GetSpeedToAddress(
		request.ServerAddress)
	if haveLinkSpeed {
		t.logFetch(request, linkSpeed)
	} else {
		if t.networkReaderContext.MaximumSpeed() < 1 {
			benchmark = enoughBytesForBenchmark(objectServer, request)
			if benchmark {
				objectServer.SetExclusiveGetObjects(true)
				t.logger.Printf("Fetch(%s) %d objects and benchmark speed\n",
					request.ServerAddress, len(request.Hashes))
			} else {
				t.logFetch(request, 0)
			}
		} else {
			t.logFetch(request, t.networkReaderContext.MaximumSpeed())
		}
	}
	objectsReader, err := objectServer.GetObjects(request.Hashes)
	if err != nil {
		t.logger.Printf("Error getting object reader: %s\n", err.Error())
		return err
	}
	defer objectsReader.Close()
	var totalLength uint64
	defer t.rescanObjectCacheFunction()
	timeStart := time.Now()
	for _, hash := range request.Hashes {
		length, reader, err := objectsReader.NextObject()
		if err != nil {
			t.logger.Println(err)
			return err
		}
		r := io.Reader(reader)
		if haveLinkSpeed {
			if linkSpeed > 0 {
				r = rateio.NewReaderContext(linkSpeed,
					uint64(t.networkReaderContext.SpeedPercent()),
					&rateio.ReadMeasurer{}).NewReader(reader)
			}
		} else if !benchmark {
			r = t.networkReaderContext.NewReader(reader)
		}
		err = readOne(t.objectsDir, hash, length, r)
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
		format.FormatBytes(totalLength), format.Duration(duration),
		format.FormatBytes(speed))
	return nil
}

func (t *rpcType) logFetch(request sub.FetchRequest, speed uint64) {
	speedString := "unlimited speed"
	if speed > 0 {
		speedString = format.FormatBytes(
			speed*uint64(t.networkReaderContext.SpeedPercent())/100) + "/s"
	}
	t.logger.Printf("Fetch(%s) %d objects at %s\n",
		request.ServerAddress, len(request.Hashes), speedString)
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
