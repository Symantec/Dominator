package filesystem

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"io"
	"os"
	"path"
	"syscall"
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
	filename := path.Join(objSrv.baseDir, objectcache.HashToFilename(hashVal))
	// Check for existing object and collision.
	fi, err := os.Lstat(filename)
	if err == nil {
		if !fi.Mode().IsRegular() {
			return hashVal, false, errors.New("Existing non-file: " + filename)
		}
		if err := collisionCheck(data, filename, fi.Size()); err != nil {
			return hashVal, false, errors.New(
				"Collision detected: " + err.Error())
		}
		// No collision and no error: it's the same object. Go home early.
		return hashVal, false, nil
	}
	if err = os.MkdirAll(path.Dir(filename), syscall.S_IRWXU); err != nil {
		return hashVal, false, err
	}
	if err := fsutil.CopyToFile(filename, filePerms, bytes.NewReader(data),
		length); err != nil {
		return hashVal, false, err
	}
	objSrv.rwLock.Lock()
	objSrv.sizesMap[hashVal] = uint64(len(data))
	objSrv.rwLock.Unlock()
	return hashVal, true, nil
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
