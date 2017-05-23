package scanner

import (
	"bufio"
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"io"
	"os"
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
	length      uint64
	totalBytes  uint64
	oldest      *unreferencedObjectsEntry
	newest      *unreferencedObjectsEntry
	hashToEntry map[hash.Hash]*unreferencedObjectsEntry
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
		return nil, err
	}
	list := &unreferencedObjectsList{
		length:      length,
		hashToEntry: make(map[hash.Hash]*unreferencedObjectsEntry, length),
	}
	for count := uint64(0); count < length; count++ {
		var object unreferencedObject
		if err := decoder.Decode(&object); err != nil {
			return nil, err
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
	length uint64) {
	if _, ok := list.hashToEntry[hashVal]; !ok {
		object := unreferencedObject{hashVal, length, time.Now()}
		list.addEntry(&unreferencedObjectsEntry{object: object})
	}
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
