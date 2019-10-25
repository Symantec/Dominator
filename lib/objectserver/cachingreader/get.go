package cachingreader

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
)

type objectsReader struct {
	downloadedObjects  uint
	downloadedBytes    uint64
	completionChannels map[<-chan struct{}]chan<- struct{}
	objectsToRead      []*objectType // nil: read from upstream, no caching.
	objectClient       *client.ObjectClient
	objectsReader      objectserver.FullObjectsReader
	objSrv             *ObjectServer
	totalBytes         uint64
	totalObjects       uint
	waitedObjects      uint
	waitedBytes        uint64
}

type readerObject struct {
	object *objectType
	file   *os.File
	objSrv *ObjectServer
}

func saveObject(filename string,
	objectsReader objectserver.ObjectsReader) error {
	if size, reader, err := objectsReader.NextObject(); err != nil {
		return err
	} else {
		defer reader.Close()
		err := fsutil.CopyToFile(filename, privateFilePerms, reader, size)
		if err == nil {
			return nil
		}
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(filename), dirPerms); err != nil {
			return err
		}
		return fsutil.CopyToFile(filename, privateFilePerms, reader, size)
	}
}

func (objSrv *ObjectServer) getObjects(hashes []hash.Hash) (
	objectserver.ObjectsReader, error) {
	or := objectsReader{
		completionChannels: make(map[<-chan struct{}]chan<- struct{},
			len(hashes)),
		objSrv:        objSrv,
		objectsToRead: make([]*objectType, len(hashes)),
	}
	var hashesToFetch []hash.Hash
	var fetchToReadIndex []int
	objSrv.rwLock.Lock()
	defer objSrv.rwLock.Unlock()
	for index, hashVal := range hashes {
		if object := objSrv.getObjectWithLock(hashVal); object == nil {
			fetchToReadIndex = append(fetchToReadIndex, index)
			hashesToFetch = append(hashesToFetch, hashVal)
		} else {
			or.objectsToRead[index] = object
		}
	}
	if len(hashesToFetch) < 1 {
		return &or, nil
	}
	or.objectClient = client.NewObjectClient(objSrv.objectServerAddress)
	if realOR, err := or.objectClient.GetObjects(hashesToFetch); err != nil {
		or.Close()
		return nil, err
	} else {
		or.objectsReader = realOR.(objectserver.FullObjectsReader)
	}
	sizes := or.objectsReader.ObjectSizes()
	for index, hashVal := range hashesToFetch {
		size := sizes[index]
		if !objSrv.releaseSpaceWithLock(size) {
			continue // Too large: read directly without caching.
		}
		if _, ok := objSrv.objects[hashVal]; ok {
			// Duplicate fetch of same hash: read directly since we're committed
			// to read it.
			or.objectsToRead[fetchToReadIndex[index]] = nil
			continue
		}
		completionChannel := make(chan struct{})
		or.completionChannels[completionChannel] = completionChannel
		object := &objectType{
			hash:               hashVal,
			size:               size,
			downloadingChannel: completionChannel,
			usageCount:         1,
		}
		or.objectsToRead[fetchToReadIndex[index]] = object
		objSrv.downloadingBytes += size
		objSrv.objects[hashVal] = object
	}
	return &or, nil
}

func (or *objectsReader) Close() error {
	or.objSrv.logger.Printf(
		"objectcache: total: %d (%s), downloaded: %d (%s), waited: %d (%s)\n",
		or.totalObjects, format.FormatBytes(or.totalBytes),
		or.downloadedObjects, format.FormatBytes(or.downloadedBytes),
		or.waitedObjects, format.FormatBytes(or.waitedBytes))
	timeoutFunction(or.objSrv.rwLock.Lock, time.Second*10)
	for _, object := range or.objectsToRead {
		if object != nil {
			or.objSrv.putObjectWithLock(object)
		}
	}
	or.objSrv.rwLock.Unlock()
	if or.objectClient == nil {
		return nil
	}
	var err error
	if e := or.objectsReader.Close(); err == nil && e != nil {
		err = e
	}
	if e := or.objectClient.Close(); err == nil && e != nil {
		err = e
	}
	if err != nil {
		return err
	}
	return nil
}

func (or *objectsReader) NextObject() (uint64, io.ReadCloser, error) {
	if len(or.objectsToRead) < 1 {
		return 0, nil, io.EOF
	}
	object := or.objectsToRead[0]
	or.objectsToRead = or.objectsToRead[1:]
	if object == nil { // No caching.
		return or.objectsReader.NextObject()
	}
	filename := filepath.Join(or.objSrv.baseDir,
		objectcache.HashToFilename(object.hash))
	or.objSrv.rwLock.RLock()
	downloadingChannel := object.downloadingChannel
	or.objSrv.rwLock.RUnlock()
	if downloadingChannel != nil {
		completionChannel := or.completionChannels[downloadingChannel]
		if completionChannel != nil { // I am the downloader.
			err := saveObject(filename, or.objectsReader)
			or.objSrv.rwLock.Lock()
			object.downloadingChannel = nil
			if err == nil {
				or.objSrv.cachedBytes += object.size
			} else {
				delete(or.objSrv.objects, object.hash)
				object.usageCount--
			}
			or.objSrv.downloadingBytes -= object.size
			or.objSrv.rwLock.Unlock()
			close(completionChannel)
			if err != nil {
				return 0, nil, err
			}
			or.downloadedObjects++
			or.downloadedBytes += object.size
		} else { // Someone else is the downloader.
			<-downloadingChannel // It's still downloading: wait.
			or.waitedObjects++
			or.waitedBytes += object.size
		}
	}
	if file, err := os.Open(filename); err != nil {
		return 0, nil, err
	} else {
		or.totalObjects++
		or.totalBytes += object.size
		return object.size, &readerObject{object, file, or.objSrv}, nil
	}
}

func (ro *readerObject) Close() error {
	err := ro.file.Close()
	ro.objSrv.rwLock.Lock()
	ro.objSrv.putObjectWithLock(ro.object)
	ro.objSrv.rwLock.Unlock()
	return err
}

func (ro *readerObject) Read(p []byte) (int, error) {
	return ro.file.Read(p)
}

func timeoutFunction(f func(), timeout time.Duration) {
	if timeout < 0 {
		f()
		return
	}
	completionChannel := make(chan struct{}, 1)
	go func() {
		f()
		completionChannel <- struct{}{}
	}()
	timer := time.NewTimer(timeout)
	select {
	case <-completionChannel:
		if !timer.Stop() {
			<-timer.C
		}
		return
	case <-timer.C:
		os.Stderr.Write([]byte("lock timeout. Full stack trace follows:\n"))
		buf := make([]byte, 1024*1024)
		nBytes := runtime.Stack(buf, true)
		os.Stderr.Write(buf[0:nBytes])
		os.Stderr.Write([]byte("\n"))
		panic("timeout")
	}
}
