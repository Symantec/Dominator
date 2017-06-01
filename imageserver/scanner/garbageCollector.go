package scanner

import (
	"bufio"
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"io"
	"os"
	"path"
	"time"
)

type unreferencedObject struct {
	Hash   hash.Hash
	Length uint64
	Age    time.Time
}

type unreferencedObjectsEntry struct {
	object unreferencedObject
	prev   *unreferencedObjectsEntry
	next   *unreferencedObjectsEntry
}

type unreferencedObjectsList struct {
	length              uint64
	totalBytes          uint64
	oldest              *unreferencedObjectsEntry
	newest              *unreferencedObjectsEntry
	hashToEntry         map[hash.Hash]*unreferencedObjectsEntry
	lastRegeneratedTime time.Time
}

func loadUnreferencedObjects(fName string) (*unreferencedObjectsList, error) {
	file, err := os.Open(fName)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		return &unreferencedObjectsList{
			hashToEntry: make(map[hash.Hash]*unreferencedObjectsEntry),
		}, nil
	}
	defer file.Close()
	reader := fsutil.NewChecksumReader(bufio.NewReader(file))
	decoder := gob.NewDecoder(reader)
	var length uint64
	if err := decoder.Decode(&length); err != nil {
		return nil, errors.New("error decoding list length: " + err.Error())
	}
	list := &unreferencedObjectsList{
		hashToEntry: make(map[hash.Hash]*unreferencedObjectsEntry, length),
	}
	for count := uint64(0); count < length; count++ {
		var object unreferencedObject
		if err := decoder.Decode(&object); err != nil {
			return nil, errors.New("error decoding object: " + err.Error())
		}
		entry := &unreferencedObjectsEntry{object: object}
		list.addEntry(entry)
	}
	if err := reader.VerifyChecksum(); err != nil {
		return nil, err
	}
	return list, nil
}

func (list *unreferencedObjectsList) write(w io.Writer) error {
	writer := fsutil.NewChecksumWriter(w)
	encoder := gob.NewEncoder(writer)
	if err := encoder.Encode(list.length); err != nil {
		return err
	}
	for entry := list.oldest; entry != nil; entry = entry.next {
		if err := encoder.Encode(entry.object); err != nil {
			return err
		}
	}
	return writer.WriteChecksum()
}

func (list *unreferencedObjectsList) addEntry(entry *unreferencedObjectsEntry) {
	entry.prev = list.newest
	if list.oldest == nil {
		list.oldest = entry
	} else {
		list.newest.next = entry
	}
	list.newest = entry
	list.hashToEntry[entry.object.Hash] = entry
	list.length++
	list.totalBytes += entry.object.Length
}

func (list *unreferencedObjectsList) addObject(hashVal hash.Hash,
	length uint64) bool {
	if _, ok := list.hashToEntry[hashVal]; !ok {
		object := unreferencedObject{hashVal, length, time.Now()}
		list.addEntry(&unreferencedObjectsEntry{object: object})
		return true
	}
	return false
}

func (list *unreferencedObjectsList) list() map[hash.Hash]uint64 {
	objectsMap := make(map[hash.Hash]uint64, list.length)
	for entry := list.oldest; entry != nil; entry = entry.next {
		objectsMap[entry.object.Hash] = entry.object.Length
	}
	return objectsMap
}

func (list *unreferencedObjectsList) removeObject(hashVal hash.Hash) bool {
	entry := list.hashToEntry[hashVal]
	if entry == nil {
		return false
	}
	if entry.prev == nil {
		list.oldest = entry.next
	} else {
		entry.prev.next = entry.next
	}
	if entry.next == nil {
		list.newest = entry.prev
	} else {
		entry.next.prev = entry.prev
	}
	list.length--
	list.totalBytes -= entry.object.Length
	return true
}

func (imdb *ImageDataBase) maybeAddToUnreferencedObjectsList(
	inodeTable filesystem.InodeTable) {
	// First get a list of objects in the image being deleted.
	objects := make(map[hash.Hash]uint64)
	for _, inode := range inodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			objects[inode.Hash] = inode.Size
		}
	}
	// Scan all remaining images and remove their objects from the list.
	for _, image := range imdb.imageMap {
		for _, inode := range image.FileSystem.InodeTable {
			if inode, ok := inode.(*filesystem.RegularInode); ok {
				delete(objects, inode.Hash)
			}
		}
	}
	changed := false
	for object, size := range objects {
		if imdb.unreferencedObjects.addObject(object, size) {
			changed = true
		}
	}
	if changed {
		imdb.saveUnreferencedObjectsList()
	}
}

func (imdb *ImageDataBase) removeFromUnreferencedObjectsList(
	inodeTable filesystem.InodeTable) {
	changed := false
	for _, inode := range inodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			if imdb.unreferencedObjects.removeObject(inode.Hash) {
				changed = true
			}
		}
	}
	if changed {
		imdb.saveUnreferencedObjectsList()
	}
}

func (imdb *ImageDataBase) saveUnreferencedObjectsList() {
	if err := imdb.writeUnreferencedObjectsList(); err != nil {
		imdb.logger.Printf("Error writing unreferenced objects list: %s\n",
			err)
	}
}

func (imdb *ImageDataBase) writeUnreferencedObjectsList() error {
	filename := path.Join(imdb.baseDir, unreferencedObjectsFile)
	file, err := fsutil.CreateRenamingWriter(filename, filePerms)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	return imdb.unreferencedObjects.write(writer)
}

func (imdb *ImageDataBase) garbageCollector(bytesToDelete uint64) (
	uint64, error) {
	if bytesToDelete < 1 {
		return 0, nil
	}
	imdb.Lock()
	firstEntry := imdb.unreferencedObjects.oldest
	entry := imdb.unreferencedObjects.oldest
	var nObjects uint64
	var nBytes uint64
	for ; entry != nil && nBytes < bytesToDelete; entry = entry.next {
		nObjects++
		nBytes += entry.object.Length
	}
	imdb.unreferencedObjects.length -= nBytes
	imdb.unreferencedObjects.totalBytes -= nObjects
	imdb.unreferencedObjects.oldest = entry
	if entry == nil {
		imdb.unreferencedObjects.newest = nil
	} else if entry.prev != nil {
		entry.prev.next = nil
		entry.prev = nil
	}
	imdb.Unlock()
	if firstEntry == nil {
		return 0, nil
	}
	nBytes = 0
	var err error
	for entry := firstEntry; entry != nil; entry = entry.next {
		if e := imdb.objectServer.DeleteObject(entry.object.Hash); e != nil {
			if err == nil {
				err = e
			}
		} else {
			nBytes += entry.object.Length
		}
	}
	imdb.saveUnreferencedObjectsList()
	return nBytes, err
}

// This grabs the lock.
func (imdb *ImageDataBase) maybeRegenerateUnreferencedObjectsList() {
	imdb.RLock()
	lastRegeneratedTime := imdb.unreferencedObjects.lastRegeneratedTime
	imdb.RUnlock()
	lastMutationTime := imdb.objectServer.LastMutationTime()
	if lastMutationTime.After(lastRegeneratedTime) {
		imdb.regenerateUnreferencedObjectsList()
	}
}

// This grabs the lock.
func (imdb *ImageDataBase) regenerateUnreferencedObjectsList() {
	scanTime := time.Now()
	// First generate list of currently unused objects.
	objectsMap := imdb.objectServer.ListObjectSizes()
	imdb.Lock()
	defer imdb.Unlock()
	for _, image := range imdb.imageMap {
		for _, inode := range image.FileSystem.InodeTable {
			if inode, ok := inode.(*filesystem.RegularInode); ok {
				delete(objectsMap, inode.Hash)
			}
		}
	}
	changed := false
	// Now add unused objects to cached list.
	for hashVal, length := range objectsMap {
		if imdb.unreferencedObjects.addObject(hashVal, length) {
			changed = true
		}
	}
	// Finally remove objects from cached list which are no longer unreferenced.
	for entry := imdb.unreferencedObjects.oldest; entry != nil; {
		hashVal := entry.object.Hash
		entry = entry.next
		if _, ok := objectsMap[hashVal]; !ok {
			if imdb.unreferencedObjects.removeObject(hashVal) {
				changed = true
			}
		}
	}
	if changed {
		imdb.saveUnreferencedObjectsList()
	}
	imdb.unreferencedObjects.lastRegeneratedTime = scanTime
}
