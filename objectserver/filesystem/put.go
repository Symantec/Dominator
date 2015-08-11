package filesystem

import (
	"bufio"
	"bytes"
	"crypto/sha512"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"os"
	"path"
)

const buflen = 65536

func (objSrv *FileSystemObjectServer) putObject(data []byte,
	expectedHash *hash.Hash) (
	hash.Hash, error) {
	var hash hash.Hash
	hasher := sha512.New()
	if hasher.Size() != len(hash) {
		return hash, errors.New("Incompatible hash size")
	}
	_, err := hasher.Write(data)
	if err != nil {
		return hash, err
	}
	copy(hash[:], hasher.Sum(nil))
	if expectedHash != nil {
		if hash != *expectedHash {
			return hash, errors.New(fmt.Sprintf(
				"Hash mismatch. Computed=%x, expected=%x", hash, *expectedHash))
		}
	}
	filename := path.Join(objSrv.topDirectory, objectcache.HashToFilename(hash))
	err = os.MkdirAll(path.Dir(filename), 0755)
	if err != nil {
		return hash, err
	}
	// Check for existing object and collision.
	fi, err := os.Lstat(filename)
	if err == nil {
		if !fi.Mode().IsRegular() {
			return hash, errors.New("Existing non-file: " + filename)
		}
		collision, err := collisionCheck(data, filename)
		if collision {
			return hash, errors.New("Collision detected: " + err.Error())
		}
		if err != nil {
			return hash, err
		}
		// No collision and no error: it's the same object. Go home early.
		return hash, nil
	}
	file, err := os.OpenFile(filename, os.O_WRONLY, 0660)
	if err != nil {
		return hash, err
	}
	defer file.Close()
	_, err = file.Write(data)
	if err != nil {
		return hash, err
	}
	return hash, nil
}

func collisionCheck(data []byte, filename string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, nil
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		return false, err
	}
	if int64(len(data)) != fi.Size() {
		return true, errors.New(fmt.Sprintf(
			"length mismatch. Data=%d, existing object=%d",
			len(data), fi.Size()))
	}
	reader := bufio.NewReader(file)
	buffer := make([]byte, 0, buflen)
	for len(data) > 0 {
		buf := buffer[:len(data)]
		nread, err := reader.Read(buf)
		if err != nil {
			return true, err
		}
		if bytes.Compare(data[:nread], buf[:nread]) != 0 {
			return true, errors.New("content mismatch")
		}
		data = data[nread:]
	}
	return false, nil
}
