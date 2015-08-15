package untar

import (
	"archive/tar"
	"github.com/Symantec/Dominator/lib/filesystem"
)

type DataHandler interface {
	HandleData(data []byte) error
}

func Decode(tarReader *tar.Reader, dataHandler DataHandler) (
	*filesystem.FileSystem, error) {
	return decode(tarReader, dataHandler)
}
