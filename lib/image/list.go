package image

import (
	"github.com/Symantec/Dominator/lib/hash"
)

func (image *Image) listObjects() []hash.Hash {
	hashes := make([]hash.Hash, 0, image.FileSystem.NumRegularInodes+2)
	image.forEachObject(func(hashVal hash.Hash) error {
		hashes = append(hashes, hashVal)
		return nil
	})
	return hashes
}
