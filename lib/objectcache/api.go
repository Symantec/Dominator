package objectcache

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type ObjectCache []hash.Hash

func ScanObjectCache(cacheDirectoryName string) (ObjectCache, error) {
	objectCache := make(ObjectCache, 0, 16)
	return scanObjectCache(cacheDirectoryName, "", objectCache)
}

func Decode(reader io.Reader) (ObjectCache, error) {
	return decode(reader)
}

func (objectCache ObjectCache) Encode(writer io.Writer) error {
	return objectCache.encode(writer)
}

func CompareObjects(left, right ObjectCache, logWriter io.Writer) bool {
	return compareObjects(left, right, logWriter)
}

func FilenameToHash(fileName string) (hash.Hash, error) {
	return filenameToHash(fileName)
}

func HashToFilename(hash hash.Hash) string {
	return hashToFilename(hash)
}

func ObjectMapToCache(objectMap map[hash.Hash]uint64) ObjectCache {
	return objectMapToCache(objectMap)
}

func ReadObject(reader io.Reader, length uint64, expectedHash *hash.Hash) (
	hash.Hash, []byte, error) {
	return readObject(reader, length, expectedHash)
}
