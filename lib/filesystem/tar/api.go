package tar

import (
	"archive/tar"
	"io"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/objectserver/client"
)

func Encode(tarWriter *tar.Writer, fileSystem *filesystem.FileSystem,
	objectClient *client.ObjectClient) error {
	return encode(tarWriter, fileSystem, objectClient)
}

func Write(writer io.Writer, fileSystem *filesystem.FileSystem,
	objectClient *client.ObjectClient) error {
	return write(writer, fileSystem, objectClient)
}
