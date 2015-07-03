package scanner

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

func filenameToBytes(fileName string) ([]byte, error) {
	bytes := make([]byte, 0, 64)
	var prev_nibble byte = 16
	for _, char := range fileName {
		var nibble byte = 16
		if char >= '0' && char <= '9' {
			nibble = byte(char) - '0'
		} else if char >= 'a' && char <= 'f' {
			nibble = byte(char) - 'a' + 10
		} else {
			continue
		}
		if prev_nibble < 16 {
			bytes = append(bytes, nibble|prev_nibble<<4)
			prev_nibble = 16
		} else {
			prev_nibble = nibble
		}
	}
	return bytes, nil
}

func addCacheEntry(fileName string, cache [][]byte) ([][]byte, error) {
	bytes, err := filenameToBytes(fileName)
	if err != nil {
		return nil, err
	}
	if len(cache) >= cap(cache) {
		newCache := make([][]byte, 0, cap(cache)*2)
		copy(newCache, cache)
		cache = newCache
	}
	return append(cache, bytes), nil
}

func scanObjectCache(cacheDirectoryName string, subpath string,
	cache [][]byte) ([][]byte, error) {
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
		err = cleanPath(cacheDirectoryName, name)
		if err != nil {
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
