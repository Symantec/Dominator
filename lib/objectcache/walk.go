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
	sort.Strings(names)
	for _, name := range names {
		lastChar := name[len(name)-1]
		if lastChar == '~' || lastChar == '^' {
			continue
		}
		pathname := path.Join(myPathName, name)
		if !validatePath(name) {
			if err := os.RemoveAll(pathname); err != nil {
				return nil, err
			}
			continue
		}
		fi, err := os.Lstat(pathname)
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
