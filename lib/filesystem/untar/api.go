package untar

import (
	"archive/tar"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type DataHandler interface {
	HandleData(reader io.Reader, length uint64) (hash.Hash, error)
}

func Decode(tarReader *tar.Reader, dataHandler DataHandler,
	filter *filter.Filter) (*filesystem.FileSystem, error) {
	return decode(tarReader, dataHandler, filter)
}
