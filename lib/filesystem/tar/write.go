package tar

import (
	"archive/tar"
	"io"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/objectserver"
)

func write(writer io.Writer, fileSystem *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter) error {
	tarWriter := tar.NewWriter(writer)
	if err := Encode(tarWriter, fileSystem, objectsGetter); err != nil {
		tarWriter.Close()
		return err
	}
	return tarWriter.Close()
}
