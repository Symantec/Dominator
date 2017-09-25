package filesystem

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
)

const (
	filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP
	buflen    = 65536
)

func (objSrv *ObjectServer) addObject(reader io.Reader, length uint64,
	expectedHash *hash.Hash) (hash.Hash, bool, error) {
	hashVal, data, err := objectcache.ReadObject(reader, length, expectedHash)
	if err != nil {
		return hashVal, false, err
	}
	length = uint64(len(data))
	filename := path.Join(objSrv.baseDir, objectcache.HashToFilename(hashVal))
	// Check for existing object and collision.
	if isNew, err := objSrv.addOrCompare(hashVal, data, filename); err != nil {
		return hashVal, false, err
	} else {
		objSrv.rwLock.Lock()
		objSrv.sizesMap[hashVal] = uint64(len(data))
		objSrv.lastMutationTime = time.Now()
		objSrv.rwLock.Unlock()
		if objSrv.addCallback != nil {
			objSrv.addCallback(hashVal, uint64(len(data)), isNew)
		}
		return hashVal, isNew, nil
	}
}

func (objSrv *ObjectServer) addOrCompare(hashVal hash.Hash, data []byte,
	filename string) (bool, error) {
	fi, err := os.Lstat(filename)
	if err == nil {
		if !fi.Mode().IsRegular() {
			return false, errors.New("existing non-file: " + filename)
		}
		if err := collisionCheck(data, filename, fi.Size()); err != nil {
			return false, errors.New("collision detected: " + err.Error())
		}
		// No collision and no error: it's the same object. Go home early.
		return false, nil
	}
	objSrv.garbageCollector()
	if err = os.MkdirAll(path.Dir(filename), syscall.S_IRWXU); err != nil {
		return false, err
	}
	if err := fsutil.CopyToFile(filename, filePerms, bytes.NewReader(data),
		uint64(len(data))); err != nil {
		return false, err
	}
	return true, nil
}

func collisionCheck(data []byte, filename string, size int64) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	if int64(len(data)) != size {
		return errors.New(fmt.Sprintf(
			"length mismatch. Data=%d, existing object=%d",
			len(data), size))
	}
	reader := bufio.NewReader(file)
	buffer := make([]byte, 0, buflen)
	for len(data) > 0 {
		numToRead := len(data)
		if numToRead > cap(buffer) {
			numToRead = cap(buffer)
		}
		buf := buffer[:numToRead]
		nread, err := reader.Read(buf)
		if err != nil {
			return err
		}
		if bytes.Compare(data[:nread], buf[:nread]) != 0 {
			return errors.New("content mismatch")
		}
		data = data[nread:]
	}
	return nil
}
