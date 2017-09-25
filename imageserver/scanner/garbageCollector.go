package scanner

import (
	"bufio"
	"encoding/gob"
	"errors"
	"io"
	"os"
	"path"
	"time"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/log"
)

var timeFormat string = "02 Jan 2006 15:04:05.99 MST"

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

func (list *unreferencedObjectsList) write(w io.Writer,
	logger log.Logger) error {
	writer := fsutil.NewChecksumWriter(w)
	encoder := gob.NewEncoder(writer)
	if err := encoder.Encode(uint64(len(list.hashToEntry))); err != nil {
		return err
	}
	nWritten := 0
	for entry := list.oldest; entry != nil; entry = entry.next {
		if err := encoder.Encode(entry.object); err != nil {
			return err
		}
		nWritten++
	}
	if len(list.hashToEntry) != nWritten {
		logger.Panicf("unref objects list length: %d != num entries: %d\n",
			len(list.hashToEntry), nWritten)
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
	list.totalBytes += entry.object.Length
}

// Returns true if object was added, else false if object was already on list.
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
	objectsMap := make(map[hash.Hash]uint64, len(list.hashToEntry))
	for entry := list.oldest; entry != nil; entry = entry.next {
		objectsMap[entry.object.Hash] = entry.object.Length
	}
	return objectsMap
}

// Returns true if object was removed, else false (if not on list).
func (list *unreferencedObjectsList) removeObject(hashVal hash.Hash) bool {
	entry := list.hashToEntry[hashVal]
	if entry == nil {
		return false
	}
	delete(list.hashToEntry, hashVal)
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
	list.totalBytes -= entry.object.Length
	return true
}

func (imdb *ImageDataBase) maybeAddToUnreferencedObjectsList(
	fs *filesystem.FileSystem) {
	// First get a list of objects in the image being deleted.
	objects := fs.GetObjects()
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
	// Run garbage collector and save new list in the background.
	unreferencedBytes := imdb.unreferencedObjects.totalBytes
	go func() {
		var bytesToDelete uint64
		maxUnrefData := uint64(*imageServerMaxUnrefData)
		if maxUnrefData > 0 && unreferencedBytes > maxUnrefData {
			bytesToDelete = unreferencedBytes - maxUnrefData
		}
		var maxAge time.Time
		if *imageServerMaxUnrefAge > 0 {
			maxAge = time.Now().Add(-*imageServerMaxUnrefAge)
		}
		if bytesToDelete > 0 || !maxAge.IsZero() {
			nDeleted, _ := imdb.collectGarbage(bytesToDelete, maxAge)
			if nDeleted > 0 {
				return // Garbage collector saved unreferenced objects list.
			}
		}
		if changed {
			imdb.saveUnreferencedObjectsList(true)
		}
	}()
}

func (imdb *ImageDataBase) removeFromUnreferencedObjectsListAndSave(
	inodeTable filesystem.InodeTable) {
	if imdb.removeFromUnreferencedObjectsList(inodeTable) {
		imdb.saveUnreferencedObjectsList(false)
	}
}

func (imdb *ImageDataBase) removeFromUnreferencedObjectsList(
	inodeTable filesystem.InodeTable) bool {
	changed := false
	for _, inode := range inodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			if imdb.unreferencedObjects.removeObject(inode.Hash) {
				changed = true
			}
		}
	}
	return changed
}

func (imdb *ImageDataBase) saveUnreferencedObjectsList(grabLock bool) {
	if grabLock {
		imdb.RLock()
		defer imdb.RUnlock()
	}
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
	return imdb.unreferencedObjects.write(writer, imdb.logger)
}

func (imdb *ImageDataBase) garbageCollector(bytesToDelete uint64) (
	uint64, error) {
	return imdb.collectGarbage(bytesToDelete, time.Time{})
}

// This grabs and releases the lock.
func (imdb *ImageDataBase) collectGarbage(bytesToDelete uint64,
	deleteBefore time.Time) (uint64, error) {
	if bytesToDelete < 1 && deleteBefore.IsZero() {
		return 0, nil
	}
	if deleteBefore.IsZero() {
		imdb.logger.Printf("Garbage collector deleting: %s\n",
			format.FormatBytes(bytesToDelete))
	} else {
		imdb.logger.Printf("Garbage collector deleting: %s or before: %s\n",
			format.FormatBytes(bytesToDelete), deleteBefore.Format(timeFormat))
	}
	imdb.Lock()
	var firstEntryToDelete *unreferencedObjectsEntry
	entry := imdb.unreferencedObjects.oldest
	var nObjectsDeleted uint64
	var nBytesDeleted uint64
	for ; entry != nil && (nBytesDeleted < bytesToDelete ||
		entry.object.Age.Before(deleteBefore)); entry = entry.next {
		firstEntryToDelete = imdb.unreferencedObjects.oldest
		delete(imdb.unreferencedObjects.hashToEntry, entry.object.Hash)
		nObjectsDeleted++
		nBytesDeleted += entry.object.Length
	}
	imdb.unreferencedObjects.totalBytes -= nBytesDeleted
	imdb.unreferencedObjects.oldest = entry
	if entry == nil {
		imdb.unreferencedObjects.newest = nil
	} else if entry.prev != nil {
		entry.prev.next = nil
		entry.prev = nil
	}
	imdb.Unlock()
	if firstEntryToDelete == nil {
		imdb.logger.Println("Garbage collector: nothing to delete")
		return 0, nil
	}
	nBytesDeleted = 0
	var err error
	for entry := firstEntryToDelete; entry != nil; entry = entry.next {
		if e := imdb.objectServer.DeleteObject(entry.object.Hash); e != nil {
			if err == nil {
				err = e
			}
		} else {
			nBytesDeleted += entry.object.Length
		}
	}
	imdb.saveUnreferencedObjectsList(true)
	imdb.logger.Printf("Garbage collector deleted: %s in: %d objects\n",
		format.FormatBytes(nBytesDeleted), nObjectsDeleted)
	return nBytesDeleted, err
}

// This grabs and releases the lock.
func (imdb *ImageDataBase) maybeRegenerateUnreferencedObjectsList() {
	imdb.RLock()
	lastRegeneratedTime := imdb.unreferencedObjects.lastRegeneratedTime
	imdb.RUnlock()
	lastMutationTime := imdb.objectServer.LastMutationTime()
	if lastMutationTime.After(lastRegeneratedTime) {
		imdb.regenerateUnreferencedObjectsList()
	}
}

// This grabs and releases the lock.
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
		imdb.saveUnreferencedObjectsList(false)
	}
	imdb.unreferencedObjects.lastRegeneratedTime = scanTime
}

// This grabs and releases the lock.
func (imdb *ImageDataBase) garbageCollectorAddCallback(hashVal hash.Hash,
	length uint64, isNew bool) {
	imdb.Lock()
	defer imdb.Unlock()
	// Move added objects (whether new or old) to the back of the unreferenced
	// list if they are unreferenced, as they are likely to be referenced soon.
	if imdb.unreferencedObjects.removeObject(hashVal) {
		imdb.unreferencedObjects.addObject(hashVal, length)
	}
}

func (imdb *ImageDataBase) periodicGarbageCollector() {
	if *imageServerMaxUnrefData < 1 && *imageServerMaxUnrefAge < 1 {
		return
	}
	maxUnrefData := uint64(*imageServerMaxUnrefData)
	for ; ; time.Sleep(time.Minute) {
		var bytesToDelete uint64
		var maxAge time.Time
		imdb.RLock()
		unreferencedBytes := imdb.unreferencedObjects.totalBytes
		if maxUnrefData > 0 && unreferencedBytes > maxUnrefData {
			bytesToDelete = unreferencedBytes - maxUnrefData
		}
		if *imageServerMaxUnrefAge > 0 &&
			imdb.unreferencedObjects.oldest != nil {
			ageLimit := time.Now().Add(-*imageServerMaxUnrefAge)
			if imdb.unreferencedObjects.oldest.object.Age.Before(ageLimit) {
				maxAge = ageLimit
			}
		}
		imdb.RUnlock()
		if bytesToDelete > 0 || !maxAge.IsZero() {
			imdb.collectGarbage(bytesToDelete, maxAge)
		}
	}
}
