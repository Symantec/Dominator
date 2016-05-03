package memory

import (
	"github.com/Symantec/Dominator/lib/hash"
)

func (objSrv *ObjectServer) listObjectSizes() map[hash.Hash]uint64 {
	objSrv.rwLock.RLock()
	defer objSrv.rwLock.RUnlock()
	sizesMap := make(map[hash.Hash]uint64, len(objSrv.objectMap))
	for hashVal, data := range objSrv.objectMap {
		sizesMap[hashVal] = uint64(len(data))
	}
	return sizesMap
}

func (objSrv *ObjectServer) listObjects() []hash.Hash {
	objSrv.rwLock.RLock()
	defer objSrv.rwLock.RUnlock()
	hashes := make([]hash.Hash, 0, len(objSrv.objectMap))
	for hashVal := range objSrv.objectMap {
		hashes = append(hashes, hashVal)
	}
	return hashes
}
