package objectclient

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

func (objQ *ObjectAdderQueue) add(reader io.Reader, length uint64) (
	hash.Hash, error) {
	var hash hash.Hash
	if length+objQ.numBytes > objQ.maxBytes {
		if err := objQ.Flush(); err != nil {
			return hash, err
		}
	}
	hasher := sha512.New()
	data := make([]byte, length)
	nRead, err := io.ReadFull(reader, data)
	if err != nil {
		return hash, err
	}
	if uint64(nRead) != length {
		return hash, errors.New(fmt.Sprintf(
			"failed to read file data, wanted: %d, got: %d bytes",
			length, nRead))
	}
	if _, err := hasher.Write(data); err != nil {
		return hash, err
	}
	copy(hash[:], hasher.Sum(nil))
	objQ.datas = append(objQ.datas, data)
	objQ.expectedHashes = append(objQ.expectedHashes, &hash)
	objQ.numBytes += uint64(len(data))
	return hash, nil
}

func (objQ *ObjectAdderQueue) flush() error {
	_, err := objQ.client.AddObjects(objQ.datas, objQ.expectedHashes)
	if err != nil {
		return errors.New("error adding objects, remote error: " + err.Error())
	}
	objQ.numBytes = 0
	objQ.datas = nil
	objQ.expectedHashes = nil
	return nil
}
