package untar

import (
	"archive/tar"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"io"
	"io/ioutil"
)

func decode(tarReader *tar.Reader, dataHandler DataHandler) (
	*filesystem.FileSystem, error) {
	var fs filesystem.FileSystem
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if (header.Typeflag == tar.TypeReg ||
			header.Typeflag == tar.TypeRegA) &&
			header.Size > 0 {
			data, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, errors.New("error reading file data" + err.Error())
			}
			if int64(len(data)) != header.Size {
				return nil, errors.New(fmt.Sprintf(
					"failed to read file data, wanted: %d, got: %d bytes",
					header.Size, len(data)))
			}
			err = dataHandler.HandleData(data)
			if err != nil {
				return nil, err
			}
		}
		err = addHeader(&fs, header)
		if err != nil {
			return nil, err
		}
	}
	return &fs, nil
}

func addHeader(fs *filesystem.FileSystem, header *tar.Header) error {
	// TODO(rgooch): Decode header and add to fs.
	return nil
}
