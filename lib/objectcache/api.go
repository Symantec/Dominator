package objectcache

type ObjectCache [][]byte

func ScanObjectCache(cacheDirectoryName string) (ObjectCache, error) {
	objectCache := make(ObjectCache, 0, 16)
	return scanObjectCache(cacheDirectoryName, "", objectCache)
}
