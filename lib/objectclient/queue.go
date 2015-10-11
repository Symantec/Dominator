package objectclient

import (
	"crypto/sha512"
	"errors"
	"github.com/Symantec/Dominator/lib/hash"
)

func (objQ *ObjectAdderQueue) add(data []byte) (hash.Hash, error) {
	var hash hash.Hash
	if uint64(len(data))+objQ.numBytes > objQ.maxBytes {
		if err := objQ.Flush(); err != nil {
			return hash, err
		}
	}
	hasher := sha512.New()
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
