package unpacker

import (
	"errors"

	"github.com/Symantec/Dominator/lib/filesystem"
)

func (u *Unpacker) getFileSystem(streamName string) (
	*filesystem.FileSystem, error) {
	u.rwMutex.RLock()
	defer u.rwMutex.RUnlock()
	streamInfo := u.getStream(streamName)
	if streamInfo == nil {
		return nil, errors.New("unknown stream")
	}
	return streamInfo.scannedFS, nil
}
