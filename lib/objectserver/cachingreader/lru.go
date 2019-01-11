package cachingreader

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
)

const (
	privateFilePerms = syscall.S_IRUSR | syscall.S_IWUSR
	filePerms        = privateFilePerms | syscall.S_IRGRP | syscall.S_IROTH
)

func (objSrv *ObjectServer) addToLruWithLock(object *objectType) {
	if object.usageCount != 0 {
		panic("object.usageCount != 0")
	}
	if objSrv.oldest == nil { // Empty list: initialise it.
		objSrv.oldest = object
	} else { // Update previous newest object.
		if objSrv.newest == nil {
			panic("LRU has oldest but not newest entry")
		}
		objSrv.newest.newer = object
	}
	object.older = objSrv.newest
	objSrv.newest = object
	objSrv.lruBytes += object.size
	select {
	case objSrv.lruUpdateNotifier <- struct{}{}:
	default:
	}
}

func (objSrv *ObjectServer) flusher(lruUpdateNotifier <-chan struct{}) {
	flushTimer := time.NewTimer(time.Minute)
	flushTimer.Stop()
	for {
		select {
		case <-lruUpdateNotifier:
			if !flushTimer.Stop() {
				select {
				case <-flushTimer.C:
				default:
				}
			}
			flushTimer.Reset(time.Minute)
		case <-flushTimer.C:
			objSrv.saveLru()
		}
	}
}

func (objSrv *ObjectServer) getObjectWithLock(hashVal hash.Hash) *objectType {
	if object, ok := objSrv.objects[hashVal]; !ok {
		return nil
	} else {
		if object.usageCount < 1 {
			objSrv.removeFromLruWithLock(object)
		}
		object.usageCount++
		return object
	}
}

func (objSrv *ObjectServer) loadLru() error {
	startTime := time.Now()
	filename := filepath.Join(objSrv.baseDir, ".lru")
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	var hashVal hash.Hash
	objSrv.lruBytes = 0
	for { // First object is newest, last object is oldest.
		if _, err := reader.Read((&hashVal)[:]); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if object, ok := objSrv.objects[hashVal]; ok {
			objSrv.lruBytes += object.size
			if objSrv.newest == nil { // Empty list: initialise it.
				objSrv.newest = object
			} else { // Make object the oldest.
				object.newer = objSrv.oldest
				objSrv.oldest.older = object
			}
			objSrv.oldest = object
		}
	}
	objSrv.logger.Printf("Loaded LRU in %s\n",
		format.Duration(time.Since(startTime)))
	return nil
}

func (objSrv *ObjectServer) putObjectWithLock(object *objectType) {
	if object.usageCount < 1 {
		panic("object.usageCount == 0")
	}
	object.usageCount--
	if object.usageCount == 0 {
		objSrv.addToLruWithLock(object)
	}
}

// Returns true if space is available.
func (objSrv *ObjectServer) releaseSpaceWithLock(size uint64) bool {
	if objSrv.cachedBytes+objSrv.downloadingBytes+size <=
		objSrv.maxCachedBytes {
		return true
	}
	if objSrv.cachedBytes-objSrv.lruBytes+objSrv.downloadingBytes+size >
		objSrv.maxCachedBytes {
		return false // No amount of deleting unused objects will help.
	}
	for objSrv.oldest != nil {
		filename := filepath.Join(objSrv.baseDir,
			objectcache.HashToFilename(objSrv.oldest.hash))
		if err := os.Remove(filename); err != nil {
			objSrv.logger.Println(err)
			return false
		}
		objSrv.removeFromLruWithLock(objSrv.oldest)
		objSrv.cachedBytes -= objSrv.oldest.size
		if objSrv.cachedBytes+objSrv.downloadingBytes+size <=
			objSrv.maxCachedBytes {
			return true
		}
	}
	panic("not enough space despite freeing unused objects")
}

func (objSrv *ObjectServer) removeFromLruWithLock(object *objectType) {
	if object.older == nil { // Object is the oldest.
		objSrv.oldest = object.newer
		if objSrv.oldest != nil {
			objSrv.oldest.older = nil
		}
	} else {
		object.older.newer = object.newer
	}
	if object.newer == nil { // Object is the newest.
		objSrv.newest = object.older
		if objSrv.newest != nil {
			objSrv.newest.newer = nil
		}
	} else {
		object.newer.older = object.older
	}
	object.newer = nil
	object.older = nil
	if objSrv.newest == nil && objSrv.oldest != nil {
		panic("LRU has oldest but not newest entry")
	}
	if objSrv.oldest == nil && objSrv.newest != nil {
		panic("LRU has newest but not oldest entry")
	}
	objSrv.lruBytes -= object.size
	select {
	case objSrv.lruUpdateNotifier <- struct{}{}:
	default:
	}
}

func (objSrv *ObjectServer) saveLru() {
	objSrv.rwLock.RLock()
	defer objSrv.rwLock.RUnlock()
	filename := filepath.Join(objSrv.baseDir, ".lru")
	writer, err := fsutil.CreateRenamingWriter(filename, filePerms)
	if err != nil {
		return
	}
	defer writer.Close()
	w := bufio.NewWriter(writer)
	defer w.Flush()
	// Write newest first, oldest last.
	for object := objSrv.newest; object != nil; object = object.older {
		if _, err := w.Write(object.hash[:]); err != nil {
			return
		}
	}
}
