package untar

import (
	"archive/tar"
	"io"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
)

type Hasher interface {
	Hash(reader io.Reader, length uint64) (hash.Hash, error)
}

func Decode(tarReader *tar.Reader, hasher Hasher, filter *filter.Filter) (
	*filesystem.FileSystem, error) {
	return decode(tarReader, hasher, filter)
}
