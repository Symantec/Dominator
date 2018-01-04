package tar

import (
	"archive/tar"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/objectserver/client"
)

func Encode(tarWriter *tar.Writer, fileSystem *filesystem.FileSystem,
	objectClient *client.ObjectClient) error {
	return encode(tarWriter, fileSystem, objectClient)
}
