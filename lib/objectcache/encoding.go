package objectcache

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

func decode(reader io.Reader) (ObjectCache, error) {
	var numObjects uint64
	if err := binary.Read(reader, binary.BigEndian, &numObjects); err != nil {
		return nil, err
	}
	objectCache := make(ObjectCache, numObjects)
	for index := uint64(0); index < numObjects; index++ {
		hash := &objectCache[index]
		nRead, err := io.ReadFull(reader, (*hash)[:])
		if err != nil {
			return nil, err
		}
		if nRead != len(*hash) {
			return nil, errors.New(fmt.Sprintf(
				"read: %d, expected: %d", nRead, len(hash)))
		}
	}
	return objectCache, nil
}

func (objectCache ObjectCache) encode(writer io.Writer) error {
	numObjects := uint64(len(objectCache))
	if err := binary.Write(writer, binary.BigEndian, numObjects); err != nil {
		return err
	}
	for _, hash := range objectCache {
		nWritten, err := writer.Write(hash[:])
		if err != nil {
			return err
		}
		if nWritten != len(hash) {
			return errors.New(fmt.Sprintf(
				"wrote: %d, expected: %d", nWritten, len(hash)))
		}
	}
	return nil
}
