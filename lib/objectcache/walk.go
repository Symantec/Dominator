package objectcache

import (
	"os"
	"path"
	"sort"
)

func validatePath(fileName string) bool {
	for _, char := range fileName {
		if (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') {
			continue
		}
		return false
	}
	return true
}

func cleanPath(directoryName string, fileName string) error {
	if !validatePath(fileName) {
		return os.RemoveAll(path.Join(directoryName, fileName))
	}
	return nil
}

func addCacheEntry(fileName string, cache ObjectCache) (ObjectCache, error) {
	hash, err := filenameToHash(fileName)
	if err != nil {
		return nil, err
	}
	return append(cache, hash), nil
}

func scanObjectCache(cacheDirectoryName string, subpath string,
	cache ObjectCache) (ObjectCache, error) {
	myPathName := path.Join(cacheDirectoryName, subpath)
	file, err := os.Open(myPathName)
	if err != nil {
		return nil, err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		if err = cleanPath(cacheDirectoryName, name); err != nil {
			return nil, err
		}
	}
	sort.Strings(names)
	for _, name := range names {
		fi, err := os.Lstat(path.Join(myPathName, name))
		if err != nil {
			continue
		}
		filename := path.Join(subpath, name)
		if fi.IsDir() {
			cache, err = scanObjectCache(cacheDirectoryName, filename, cache)
			if err != nil {
				return nil, err
			}
		} else {
			cache, err = addCacheEntry(filename, cache)
			if err != nil {
				return nil, err
			}
		}
	}
	return cache, nil
}
