package unpacker

import (
	"bytes"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/json"
	"path"
	"syscall"
)

const filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP |
	syscall.S_IROTH

func (u *Unpacker) writeState() error {
	buffer := new(bytes.Buffer)
	u.rwMutex.RLock()
	err := json.WriteWithIndent(buffer, "    ", u.pState)
	u.rwMutex.RUnlock()
	if err != nil {
		return err
	}
	return fsutil.CopyToFile(path.Join(u.baseDir, stateFile), filePerms, buffer,
		uint64(buffer.Len()))
}

func (u *Unpacker) writeStateWithLock() error {
	buffer := new(bytes.Buffer)
	if err := json.WriteWithIndent(buffer, "    ", u.pState); err != nil {
		return err
	}
	return fsutil.CopyToFile(path.Join(u.baseDir, stateFile), filePerms, buffer,
		uint64(buffer.Len()))
}
