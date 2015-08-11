package filesystem

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"os"
	"path"
)

func (objSrv *FileSystemObjectServer) putObject(data []byte,
	expectedHash *hash.Hash) (
	hash.Hash, error) {
	var hash hash.Hash
	hasher := sha512.New()
	if hasher.Size() != len(hash) {
		return hash, errors.New("Incompatible hash size")
	}
	_, err := hasher.Write(data)
	if err != nil {
		return hash, err
	}
	copy(hash[:], hasher.Sum(nil))
	if expectedHash != nil {
		if hash != *expectedHash {
			return hash, errors.New(fmt.Sprintf(
				"Hash mismatch. Computed=%x, expected=%x", hash, *expectedHash))
		}
	}
	filename := path.Join(objSrv.topDirectory, objectcache.HashToFilename(hash))
	err = os.MkdirAll(path.Dir(filename), 0755)
	if err != nil {
		return hash, err
	}
	file, err := os.OpenFile(filename, os.O_WRONLY, 0660)
	if err != nil {
		return hash, err
	}
	defer file.Close()
	_, err = file.Write(data)
	if err != nil {
		return hash, err
	}
	return hash, nil
}
