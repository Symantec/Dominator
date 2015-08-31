package rpcd

import (
	"bufio"
	"errors"
	"fmt"
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

func (t *rpcType) Fetch(request sub.FetchRequest,
	reply *sub.FetchResponse) error {
	rwLock.Lock()
	defer rwLock.Unlock()
	if fetchInProgress {
		return errors.New("fetch already in progress")
	}
	if updateInProgress {
		return errors.New("update in progress")
	}
	fetchInProgress = true
	go doFetch(request)
	return nil
}

func doFetch(request sub.FetchRequest) {
	defer clearFetchInProgress()
	objectServer := objectclient.NewObjectClient(request.ServerAddress)
	benchmark := false
	if networkReaderContext.MaximumSpeed() < 1 {
		benchmark = enoughBytesForBenchmark(objectServer, request)
		if benchmark {
			objectServer.SetExclusiveGetObjects(true)
		}
	}
	objectsReader, err := objectServer.GetObjects(request.Hashes)
	if err != nil {
		fmt.Printf("Error getting object reader:\t%s\n", err.Error())
		return
	}
	var totalLength uint64
	timeStart := time.Now()
	for _, hash := range request.Hashes {
		length, reader, err := objectsReader.NextObject()
		if err != nil {
			fmt.Println(err)
			return
		}
		err = readOne(hash, networkReaderContext.NewReader(reader))
		reader.Close()
		if err != nil {
			fmt.Println(err)
			return
		}
		totalLength += length
	}
	if benchmark {
		duration := time.Since(timeStart)
		speed := uint64(float64(totalLength) / duration.Seconds())
		file, err := os.Create(netbenchFilename)
		if err == nil {
			fmt.Fprintf(file, "%d\n", speed)
			file.Close()
		}
		networkReaderContext.InitialiseMaximumSpeed(speed)
	}
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

func readOne(hash hash.Hash, reader io.Reader) error {
	fmt.Printf("Reading: %x\n", hash) // TODO(rgooch): Remove debugging output.
	filename := path.Join(objectsDir, objectcache.HashToFilename(hash))
	dirname := path.Dir(filename)
	err := os.MkdirAll(dirname, syscall.S_IRUSR|syscall.S_IWUSR)
	if err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	_, err = io.Copy(writer, reader)
	if err != nil {
		return errors.New(fmt.Sprintf("error copying: %s", err.Error()))
	}
	return nil
}

func clearFetchInProgress() {
	rwLock.Lock()
	defer rwLock.Unlock()
	fetchInProgress = false
}
