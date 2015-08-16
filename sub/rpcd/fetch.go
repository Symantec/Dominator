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
	"net/rpc"
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
	client, err := rpc.DialHTTP("tcp", request.ServerAddress)
	if err != nil {
		fmt.Printf("Error dialing\t%s\n", err)
		return
	}
	objectServer := objectclient.NewObjectClient(client)
	objectsReader, err := objectServer.GetObjects(request.Hashes)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, hash := range request.Hashes {
		_, reader, err := objectsReader.NextObject()
		if err != nil {
			fmt.Println(err)
			return
		}
		err = readOne(hash, reader)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	// TODO(rgooch): Find some way to invalidate object cache and splice a new
	//               one into the current FS state and affect the current scan.
}

func readOne(hash hash.Hash, reader io.ReadCloser) error {
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
