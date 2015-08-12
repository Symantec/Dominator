package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"io"
	"os"
	"path"
)

func (objSrv *FileSystemObjectServer) getObjectReader(hash hash.Hash) (uint64,
	io.ReadCloser, error) {
	filename := path.Join(objSrv.baseDir, objectcache.HashToFilename(hash))
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
