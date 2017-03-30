package objectcache

import (
	"github.com/Symantec/Dominator/lib/hash"
)

func objectMapToCache(objectMap map[hash.Hash]uint64) ObjectCache {
	objectCache := make(ObjectCache, 0, len(objectMap))
	for hashVal := range objectMap {
		objectCache = append(objectCache, hashVal)
	}
	return objectCache
}
