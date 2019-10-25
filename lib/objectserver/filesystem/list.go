package filesystem

import (
	"github.com/Cloud-Foundations/Dominator/lib/hash"
)

func (objSrv *ObjectServer) listObjectSizes() map[hash.Hash]uint64 {
	objSrv.rwLock.RLock()
	defer objSrv.rwLock.RUnlock()
	sizesMap := make(map[hash.Hash]uint64, len(objSrv.sizesMap))
	for hashVal, size := range objSrv.sizesMap {
		sizesMap[hashVal] = uint64(size)
	}
	return sizesMap
}

func (objSrv *ObjectServer) listObjects() []hash.Hash {
	objSrv.rwLock.RLock()
	defer objSrv.rwLock.RUnlock()
	hashes := make([]hash.Hash, 0, len(objSrv.sizesMap))
	for hashVal := range objSrv.sizesMap {
		hashes = append(hashes, hashVal)
	}
	return hashes
}
