package objectserver

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/Symantec/Dominator/lib/hash"
)

const (
	privateFilePerms = syscall.S_IRUSR | syscall.S_IWUSR
)

func copyObject(filename string, objectsGetter ObjectsGetter,
	hashVal hash.Hash) error {
	size, reader, err := getObject(objectsGetter, hashVal)
	if err != nil {
		return err
	}
	defer reader.Close()
	iLength := int64(size)
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY,
		privateFilePerms)
	if err != nil {
		return err
	}
	defer file.Close()
	nCopied, err := io.CopyN(file, reader, iLength)
	if err != nil {
		os.Remove(filename)
		return err
	}
	if nCopied != iLength {
		os.Remove(filename)
		return fmt.Errorf("copied: %d, expected: %d", nCopied, iLength)
	}
	return nil
}

func getObject(objSrv ObjectsGetter, hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	hashes := make([]hash.Hash, 1)
	hashes[0] = hashVal
	objectsReader, err := objSrv.GetObjects(hashes)
	if err != nil {
		return 0, nil, err
	}
	defer objectsReader.Close()
	size, reader, err := objectsReader.NextObject()
	if err != nil {
		return 0, nil, err
	}
	return size, reader, nil
}

func linkObject(filename string, objectsGetter ObjectsGetter,
	hashVal hash.Hash) (bool, error) {
	if objectLinker, ok := objectsGetter.(ObjectLinker); ok {
		return objectLinker.LinkObject(filename, hashVal)
	}
	return false, copyObject(filename, objectsGetter, hashVal)
}
