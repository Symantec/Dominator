package image

import (
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
)

func (image *Image) listObjects() []hash.Hash {
	hashes := make([]hash.Hash, 0, image.FileSystem.NumRegularInodes+2)
	image.forEachObject(func(hashVal hash.Hash) error {
		hashes = append(hashes, hashVal)
		return nil
	})
	return hashes
}

func (image *Image) listMissingObjects(
	objectsChecker objectserver.ObjectsChecker) ([]hash.Hash, error) {
	hashBuffer := make([]hash.Hash, 1024)
	var missingObjects []hash.Hash
	index := 0
	err := image.forEachObject(func(hashVal hash.Hash) error {
		hashBuffer[index] = hashVal
		index++
		if index < len(hashBuffer) {
			return nil
		}
		var err error
		missingObjects, err = listMissingObjects(missingObjects, hashBuffer,
			objectsChecker)
		if err != nil {
			return err
		}
		index = 0
		return nil
	})
	if err != nil {
		return nil, err
	}
	return listMissingObjects(missingObjects, hashBuffer[:index],
		objectsChecker)
}

func listMissingObjects(missingObjects, hashes []hash.Hash,
	objectsChecker objectserver.ObjectsChecker) ([]hash.Hash, error) {
	objectSizes, err := objectsChecker.CheckObjects(hashes)
	if err != nil {
		return nil, err
	}
	for index, size := range objectSizes {
		if size < 1 {
			missingObjects = append(missingObjects, hashes[index])
		}
	}
	return missingObjects, nil
}
