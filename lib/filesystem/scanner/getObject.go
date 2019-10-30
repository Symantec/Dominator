package scanner

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
)

func (fs *FileSystem) getObject(hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	inodes, ok := fs.HashToInodesTable()[hashVal]
	if !ok {
		return 0, nil, fmt.Errorf("object not found: %v\n", hashVal)
	}
	filename := path.Join(fs.rootDirectoryName,
		fs.InodeToFilenamesTable()[inodes[0]][0])
	file, err := os.Open(filename)
	if err != nil {
		return 0, nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		file.Close()
		return 0, nil, err
	}
	return uint64(fi.Size()), file, nil
}
