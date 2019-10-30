package util

import (
	"bytes"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
)

type combinedObjectsGetter struct {
	filenameToHash map[string]hash.Hash
	hashToData     map[hash.Hash][]byte
	hashToFile     map[hash.Hash]string
	hashToSize     map[hash.Hash]uint64
	objectsGetter  objectserver.ObjectsGetter
}

type combinedObjectsReader struct {
	hashes        []hash.Hash
	objectsGetter *combinedObjectsGetter
	objectsReader objectserver.ObjectsReader
}

func replaceComputedFiles(fs *filesystem.FileSystem,
	computedFilesData *ComputedFilesData,
	objectsGetter objectserver.ObjectsGetter) (
	objectserver.ObjectsGetter, error) {
	newObjectsGetter, err := makeCombinedObjectsGetter(computedFilesData,
		objectsGetter)
	if err != nil {
		return nil, err
	}
	err = replaceInDirectory(fs, &fs.DirectoryInode, "/", newObjectsGetter)
	if err != nil {
		return nil, err
	}
	return newObjectsGetter, nil
}

func replaceInDirectory(fs *filesystem.FileSystem,
	directory *filesystem.DirectoryInode, dirname string,
	objectsGetter *combinedObjectsGetter) error {
	for _, entry := range directory.EntryList {
		gInode := entry.Inode()
		if inode, ok := gInode.(*filesystem.DirectoryInode); ok {
			err := replaceInDirectory(fs, inode, path.Join(dirname, entry.Name),
				objectsGetter)
			if err != nil {
				return err
			}
		} else if inode, ok := gInode.(*filesystem.ComputedRegularInode); ok {
			filename := path.Join(dirname, entry.Name)
			hashVal, ok := objectsGetter.filenameToHash[filename]
			if !ok {
				return fmt.Errorf("missing computed file: %s", filename)
			}
			fInode := &filesystem.RegularInode{
				Mode:         inode.Mode,
				Uid:          inode.Uid,
				Gid:          inode.Gid,
				MtimeSeconds: time.Now().Unix(),
				Size:         objectsGetter.hashToSize[hashVal],
				Hash:         hashVal,
			}
			entry.SetInode(fInode)
			fs.InodeTable[entry.InodeNumber] = fInode
		}
	}
	return nil
}

func makeCombinedObjectsGetter(computedFilesData *ComputedFilesData,
	objectsGetter objectserver.ObjectsGetter) (*combinedObjectsGetter, error) {
	newObjectsGetter := &combinedObjectsGetter{
		filenameToHash: make(map[string]hash.Hash),
		hashToData:     make(map[hash.Hash][]byte),
		hashToFile:     make(map[hash.Hash]string),
		hashToSize:     make(map[hash.Hash]uint64),
		objectsGetter:  objectsGetter,
	}
	for filename, data := range computedFilesData.FileData {
		checksummer := sha512.New()
		if _, err := checksummer.Write(data); err != nil {
			return nil, err
		}
		var hashVal hash.Hash
		copy(hashVal[:], checksummer.Sum(nil))
		newObjectsGetter.filenameToHash[filename] = hashVal
		newObjectsGetter.hashToData[hashVal] = data
		newObjectsGetter.hashToSize[hashVal] = uint64(len(data))
	}
	if computedFilesData.RootDirectory != "" {
		err := newObjectsGetter.scanDirectory(computedFilesData.RootDirectory,
			"/")
		if err != nil {
			return nil, err
		}
	}
	return newObjectsGetter, nil
}

func (objectsGetter *combinedObjectsGetter) GetObjects(hashes []hash.Hash) (
	objectserver.ObjectsReader, error) {
	unknownHashes := make([]hash.Hash, 0, len(hashes))
	for _, hashVal := range hashes {
		if _, ok := objectsGetter.hashToData[hashVal]; ok {
			continue
		}
		if _, ok := objectsGetter.hashToFile[hashVal]; ok {
			continue
		}
		unknownHashes = append(unknownHashes, hashVal)
	}
	objectsReader, err := objectsGetter.objectsGetter.GetObjects(unknownHashes)
	if err != nil {
		return nil, err
	}
	return &combinedObjectsReader{hashes, objectsGetter, objectsReader}, nil
}

func (objectsGetter *combinedObjectsGetter) scanDirectory(realDir string,
	mapDir string) error {
	file, err := os.Open(realDir)
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		realPath := path.Join(realDir, name)
		mapPath := path.Join(mapDir, name)
		if err := objectsGetter.scanEntry(realPath, mapPath); err != nil {
			return err
		}
	}
	return nil
}

func (objectsGetter *combinedObjectsGetter) scanEntry(realFilename string,
	mapFilename string) error {
	if fi, err := os.Lstat(realFilename); err != nil {
		return err
	} else if fi.IsDir() {
		return objectsGetter.scanDirectory(realFilename, mapFilename)
	} else if fi.Mode()&os.ModeType != 0 {
		return errors.New(realFilename + ": is not a directory or regular file")
	}
	if file, err := os.Open(realFilename); err != nil {
		return err
	} else {
		defer file.Close()
		checksummer := sha512.New()
		nWritten, err := io.Copy(checksummer, file)
		if err != nil {
			return err
		}
		var hashVal hash.Hash
		copy(hashVal[:], checksummer.Sum(nil))
		objectsGetter.filenameToHash[mapFilename] = hashVal
		objectsGetter.hashToFile[hashVal] = realFilename
		objectsGetter.hashToSize[hashVal] = uint64(nWritten)
	}
	return nil
}

func (objectsReader *combinedObjectsReader) Close() error {
	return objectsReader.objectsReader.Close()
}

func (objectsReader *combinedObjectsReader) NextObject() (
	uint64, io.ReadCloser, error) {
	hashVal := objectsReader.hashes[0]
	objectsReader.hashes = objectsReader.hashes[1:]
	if data, ok := objectsReader.objectsGetter.hashToData[hashVal]; ok {
		return objectsReader.objectsGetter.hashToSize[hashVal],
			ioutil.NopCloser(bytes.NewReader(data)), nil
	}
	if filename, ok := objectsReader.objectsGetter.hashToFile[hashVal]; ok {
		if file, err := os.Open(filename); err != nil {
			return 0, nil, err
		} else {
			return objectsReader.objectsGetter.hashToSize[hashVal], file, nil
		}
	}
	return objectsReader.objectsReader.NextObject()
}
