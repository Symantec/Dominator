package objectcache

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"io"
	"io/ioutil"
)

func readObject(reader io.Reader, length uint64, expectedHash *hash.Hash) (
	hash.Hash, []byte, error) {
	var hashVal hash.Hash
	var data []byte
	var err error
	if length < 1 {
		data, err = ioutil.ReadAll(reader)
		if err != nil {
			return hashVal, nil, err
		}
		if len(data) < 1 {
			return hashVal, nil, errors.New("zero length object cannot be added")
		}
	} else {
		data = make([]byte, length)
		nRead, err := io.ReadFull(reader, data)
		if err != nil {
			return hashVal, nil, err
		}
		if uint64(nRead) != length {
			return hashVal, nil, fmt.Errorf(
				"failed to read data, wanted: %d, got: %d bytes", length, nRead)
		}
	}
	hasher := sha512.New()
	if hasher.Size() != len(hashVal) {
		return hashVal, nil, errors.New("incompatible hash size")
	}
	if _, err := hasher.Write(data); err != nil {
		return hashVal, nil, err
	}
	copy(hashVal[:], hasher.Sum(nil))
	if expectedHash != nil {
		if hashVal != *expectedHash {
			return hashVal, nil, fmt.Errorf(
				"hash mismatch. Computed=%x, expected=%x",
				hashVal, *expectedHash)
		}
	}
	return hashVal, data, nil
}
