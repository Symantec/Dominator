package objectclient

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
)

func (objQ *ObjectAdderQueue) add(data []byte) error {
	if uint64(len(data))+objQ.numBytes > objQ.maxBytes {
		err := objQ.Flush()
		if err != nil {
			return err
		}
	}
	var hash hash.Hash
	hasher := sha512.New()
	_, err := hasher.Write(data)
	if err != nil {
		return err
	}
	copy(hash[:], hasher.Sum(nil))
	objQ.datas = append(objQ.datas, data)
	objQ.expectedHashes = append(objQ.expectedHashes, &hash)
	objQ.numBytes += uint64(len(data))
	return nil
}

func (objQ *ObjectAdderQueue) flush() error {
	// TODO(rgooch): Remove debugging output.
	fmt.Printf("Flushing: %d objects\n", len(objQ.datas))
	_, err := objQ.client.AddObjects(objQ.datas, objQ.expectedHashes)
	if err != nil {
		return errors.New("error adding objects, remote error: " + err.Error())
	}
	objQ.numBytes = 0
	objQ.datas = nil
	objQ.expectedHashes = nil
	return nil
}
