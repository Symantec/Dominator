package memory

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
)

func (objSrv *ObjectServer) addObject(reader io.Reader, length uint64,
	expectedHash *hash.Hash) (hash.Hash, bool, error) {
	hashVal, data, err := objectcache.ReadObject(reader, length, expectedHash)
	if err != nil {
		return hashVal, false, err
	}
	// Check for existing object and collision.
	objSrv.rwLock.RLock()
	oldData, ok := objSrv.objectMap[hashVal]
	objSrv.rwLock.RUnlock()
	if ok {
		if err := collisionCheck(data, oldData); err != nil {
			return hashVal, false, errors.New(
				"collision detected: " + err.Error())
		}
		// No collision and no error: it's the same object. Go home early.
		return hashVal, false, nil
	}
	objSrv.rwLock.Lock()
	objSrv.objectMap[hashVal] = data
	objSrv.rwLock.Unlock()
	return hashVal, true, nil
}

func collisionCheck(data []byte, oldData []byte) error {
	if len(data) != len(oldData) {
		return fmt.Errorf("length mismatch. Data=%d, existing object=%d",
			len(data), len(oldData))
	}
	if !bytes.Equal(data, oldData) {
		return errors.New("content mismatch")
	}
	return nil
}
