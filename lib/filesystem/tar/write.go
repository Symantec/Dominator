package tar

import (
	"archive/tar"
	"io"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/objectserver/client"
)

func write(writer io.Writer, fileSystem *filesystem.FileSystem,
	objectClient *client.ObjectClient) error {
	tarWriter := tar.NewWriter(writer)
	if err := Encode(tarWriter, fileSystem, objectClient); err != nil {
		tarWriter.Close()
		return err
	}
	return tarWriter.Close()
}
