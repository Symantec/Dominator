package untar

import (
	"archive/tar"
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/filter"
	"github.com/Cloud-Foundations/Dominator/lib/hash"
)

type Hasher interface {
	Hash(reader io.Reader, length uint64) (hash.Hash, error)
}

func Decode(tarReader *tar.Reader, hasher Hasher, filter *filter.Filter) (
	*filesystem.FileSystem, error) {
	return decode(tarReader, hasher, filter)
}
