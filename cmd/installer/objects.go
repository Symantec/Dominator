// +build linux

package main

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/wsyscall"
)

type objectsCache struct {
	bytesScanned uint64
	objects      map[hash.Hash]uint64
}

type objectsReader struct {
	cache  *objectsCache
	hashes []hash.Hash
}

func hashFile(filename string) (hash.Hash, uint64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return hash.Hash{}, 0, err
	}
	defer file.Close()
	hasher := sha512.New()
	nCopied, err := io.Copy(hasher, file)
	if err != nil {
		return hash.Hash{}, 0, err
	}
	var hashVal hash.Hash
	copy(hashVal[:], hasher.Sum(nil))
	return hashVal, uint64(nCopied), nil
}

func (cache *objectsCache) computeMissing(
	requiredObjects map[hash.Hash]uint64) (
	map[hash.Hash]uint64, uint64, uint64) {
	var requiredBytes, presentBytes uint64
	missingObjects := make(map[hash.Hash]uint64, len(requiredObjects))
	for hashVal, requiredSize := range requiredObjects {
		requiredBytes += requiredSize
		if size, ok := cache.objects[hashVal]; ok {
			presentBytes += size
		} else {
			missingObjects[hashVal] = requiredSize
		}
	}
	return missingObjects, requiredBytes, presentBytes
}

func createObjectsCache(requiredObjects map[hash.Hash]uint64,
	objGetter objectserver.ObjectsGetter, rootDevice string,
	logger log.DebugLogger) (*objectsCache, error) {
	cache := &objectsCache{objects: make(map[hash.Hash]uint64)}
	if fi, err := os.Stat(*objectsDirectory); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		logger.Debugln(0, "scanning root")
		cache.bytesScanned = 0
		startTime := time.Now()
		if err := cache.scanRoot(requiredObjects); err != nil {
			return nil, err
		}
		duration := time.Since(startTime)
		logger.Debugf(0, "scanned root %s in %s (%s/s)\n",
			format.FormatBytes(cache.bytesScanned), format.Duration(duration),
			format.FormatBytes(
				uint64(float64(cache.bytesScanned)/duration.Seconds())))
	} else if !fi.IsDir() {
		return nil,
			fmt.Errorf("%s exists but is not a directory", *objectsDirectory)
	} else {
		if err := cache.scanCache(*objectsDirectory, ""); err != nil {
			return nil, err
		}
	}
	missingObjects, requiredBytes, presentBytes := cache.computeMissing(
		requiredObjects)
	if len(missingObjects) < 1 {
		logger.Debugln(0, "object cache already has all required objects")
		return cache, nil
	}
	logger.Debugf(0, "object cache already has %d/%d objects (%s/%s)\n",
		len(cache.objects), len(requiredObjects),
		format.FormatBytes(presentBytes), format.FormatBytes(requiredBytes))
	err := cache.findAndScanUntrusted(missingObjects, rootDevice, logger)
	if err != nil {
		return nil, err
	}
	err = cache.downloadMissing(requiredObjects, objGetter, logger)
	if err != nil {
		return nil, err
	}
	return cache, nil
}

func (cache *objectsCache) downloadMissing(requiredObjects map[hash.Hash]uint64,
	objGetter objectserver.ObjectsGetter, logger log.DebugLogger) error {
	missingObjects, _, _ := cache.computeMissing(requiredObjects)
	if len(missingObjects) < 1 {
		return nil
	}
	hashes := make([]hash.Hash, 0, len(missingObjects))
	var totalBytes uint64
	for hashVal, size := range missingObjects {
		hashes = append(hashes, hashVal)
		totalBytes += size
	}
	startTime := time.Now()
	objectsReader, err := objGetter.GetObjects(hashes)
	if err != nil {
		return err
	}
	defer objectsReader.Close()
	for _, hashVal := range hashes {
		if err := cache.getNextObject(hashVal, objectsReader); err != nil {
			return err
		}
	}
	duration := time.Since(startTime)
	logger.Debugf(0, "downloaded %d objects (%s) in %s (%s/s)\n",
		len(missingObjects), format.FormatBytes(totalBytes),
		format.Duration(duration),
		format.FormatBytes(uint64(float64(totalBytes)/duration.Seconds())))
	return nil
}

func (cache *objectsCache) findAndScanUntrusted(
	requiredObjects map[hash.Hash]uint64, rootDevice string,
	logger log.DebugLogger) error {
	if err := mount(rootDevice, *mountPoint, "ext4", logger); err != nil {
		return nil
	}
	defer syscall.Unmount(*mountPoint, 0)
	logger.Debugln(0, "scanning old root")
	cache.bytesScanned = 0
	startTime := time.Now()
	foundObjects := make(map[hash.Hash]uint64)
	err := cache.scanTree(*mountPoint, true, requiredObjects, foundObjects)
	if err != nil {
		return err
	}
	var requiredBytes, foundBytes uint64
	for _, size := range requiredObjects {
		requiredBytes += size
	}
	for _, size := range foundObjects {
		foundBytes += size
	}
	duration := time.Since(startTime)
	logger.Debugf(0, "scanned old root %s in %s (%s/s)\n",
		format.FormatBytes(cache.bytesScanned), format.Duration(duration),
		format.FormatBytes(
			uint64(float64(cache.bytesScanned)/duration.Seconds())))
	logger.Debugf(0, "found %d/%d objects (%s/%s) in old file-system in %s\n",
		len(foundObjects), len(requiredObjects),
		format.FormatBytes(foundBytes), format.FormatBytes(requiredBytes),
		format.Duration(duration))
	return nil
}

func (cache *objectsCache) GetObjects(hashes []hash.Hash) (
	objectserver.ObjectsReader, error) {
	return &objectsReader{cache, hashes}, nil
}

func (cache *objectsCache) getNextObject(hashVal hash.Hash,
	objectsReader objectserver.ObjectsReader) error {
	size, reader, err := objectsReader.NextObject()
	if err != nil {
		return err
	}
	hashName := filepath.Join(*objectsDirectory,
		objectcache.HashToFilename(hashVal))
	if err := os.MkdirAll(filepath.Dir(hashName), fsutil.DirPerms); err != nil {
		return err
	}
	defer reader.Close()
	writer, err := os.Create(hashName)
	if err != nil {
		return err
	}
	defer writer.Close()
	if _, err := io.Copy(writer, reader); err != nil {
		return err
	}
	cache.objects[hashVal] = size
	return nil
}

func (cache *objectsCache) handleFile(filename string, copy bool,
	requiredObjects, foundObjects map[hash.Hash]uint64) error {
	if hashVal, size, err := hashFile(filename); err != nil {
		return err
	} else if size < 1 {
		return nil
	} else {
		cache.bytesScanned += size
		if _, ok := cache.objects[hashVal]; ok {
			return nil
		}
		if _, ok := requiredObjects[hashVal]; !ok {
			return nil
		}
		cache.objects[hashVal] = size
		if foundObjects != nil {
			foundObjects[hashVal] = size
		}
		hashName := filepath.Join(*objectsDirectory,
			objectcache.HashToFilename(hashVal))
		err := os.MkdirAll(filepath.Dir(hashName), fsutil.DirPerms)
		if err != nil {
			return err
		}
		if copy {
			reader, err := os.Open(filename)
			if err != nil {
				return err
			}
			defer reader.Close()
			writer, err := os.Create(hashName)
			if err != nil {
				return err
			}
			defer writer.Close()
			if _, err := io.Copy(writer, reader); err != nil {
				return err
			}
			return nil
		}
		return os.Symlink(filename, hashName)
	}
}

func (cache *objectsCache) scanCache(topDir, subpath string) error {
	myPathName := filepath.Join(topDir, subpath)
	file, err := os.Open(myPathName)
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		pathname := filepath.Join(myPathName, name)
		fi, err := os.Stat(pathname)
		if err != nil {
			return err
		}
		filename := filepath.Join(subpath, name)
		if fi.IsDir() {
			if err := cache.scanCache(topDir, filename); err != nil {
				return err
			}
		} else {
			hashVal, err := objectcache.FilenameToHash(filename)
			if err != nil {
				return err
			}
			cache.objects[hashVal] = uint64(fi.Size())
		}
	}
	return nil
}

func (cache *objectsCache) scanRoot(
	requiredObjects map[hash.Hash]uint64) error {
	if err := os.Mkdir(*objectsDirectory, fsutil.DirPerms); err != nil {
		return err
	}
	err := wsyscall.Mount("none", *objectsDirectory, "tmpfs", 0, "")
	if err != nil {
		return err
	}
	if err := cache.scanTree("/", false, requiredObjects, nil); err != nil {
		return err
	}
	return nil
}

func (cache *objectsCache) scanTree(topDir string, copy bool,
	requiredObjects, foundObjects map[hash.Hash]uint64) error {
	var rootStat syscall.Stat_t
	if err := syscall.Lstat(topDir, &rootStat); err != nil {
		return err
	}
	return cache.walk(topDir, rootStat.Dev, copy, requiredObjects, foundObjects)
}

func (cache *objectsCache) walk(dirname string, device uint64, copy bool,
	requiredObjects, foundObjects map[hash.Hash]uint64) error {
	file, err := os.Open(dirname)
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		pathname := filepath.Join(dirname, name)
		var stat syscall.Stat_t
		err := syscall.Lstat(pathname, &stat)
		if err != nil {
			return err
		}
		if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
			if stat.Dev != device {
				continue
			}
			err := cache.walk(pathname, device, copy, requiredObjects,
				foundObjects)
			if err != nil {
				return err
			}
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFREG {
			err := cache.handleFile(pathname, copy, requiredObjects,
				foundObjects)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (or *objectsReader) Close() error {
	return nil
}

func (or *objectsReader) NextObject() (uint64, io.ReadCloser, error) {
	if len(or.hashes) < 1 {
		return 0, nil, errors.New("all objects have been consumed")
	}
	hashVal := or.hashes[0]
	or.hashes = or.hashes[1:]
	hashName := filepath.Join(*objectsDirectory,
		objectcache.HashToFilename(hashVal))
	if file, err := os.Open(hashName); err != nil {
		return 0, nil, err
	} else {
		return or.cache.objects[hashVal], file, nil
	}
}
