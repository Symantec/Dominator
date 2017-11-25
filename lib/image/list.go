package image

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectserver"
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
	// TODO(rgooch): Implement an API that avoids copying hash lists.
	hashes := image.ListObjects()
	objectSizes, err := objectsChecker.CheckObjects(hashes)
	if err != nil {
		return nil, err
	}
	var missingObjects []hash.Hash
	for index, size := range objectSizes {
		if size < 1 {
			missingObjects = append(missingObjects, hashes[index])
		}
	}
	return missingObjects, nil
}
