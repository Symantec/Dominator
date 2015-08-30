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
	objectsReader, err := objectServer.GetObjects(request.Hashes)
	if err != nil {
		fmt.Printf("Error getting object reader:\t%s\n", err.Error())
		return
	}
	for _, hash := range request.Hashes {
		_, reader, err := objectsReader.NextObject()
		if err != nil {
			fmt.Println(err)
			return
		}
		// TODO(rgooch): Wrap reader with networkReaderContext.NewReader() once
		//               streaming RPCs are implemented.
		err = readOne(hash, reader)
		reader.Close()
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	rescanObjectCacheChannel <- true
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
