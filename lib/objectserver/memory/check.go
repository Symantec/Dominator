package memory

import (
	"github.com/Symantec/Dominator/lib/hash"
)

func (objSrv *ObjectServer) checkObjects(hashes []hash.Hash) ([]uint64, error) {
	sizesList := make([]uint64, len(hashes))
	objSrv.rwLock.RLock()
	defer objSrv.rwLock.RUnlock()
	for index, hashVal := range hashes {
		if data, ok := objSrv.objectMap[hashVal]; ok {
			sizesList[index] = uint64(len(data))
		}
	}
	return sizesList, nil
}
