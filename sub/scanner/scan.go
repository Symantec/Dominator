package scanner

import (
	"github.com/Symantec/Dominator/sub/fsrateio"
	"syscall"
)

func scanFileSystem(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext) (*FileSystem, error) {
	var fileSystem FileSystem
	fileSystem.ctx = ctx
	fileSystem.name = rootDirectoryName
	var stat syscall.Stat_t
	err := syscall.Lstat(rootDirectoryName, &stat)
	if err != nil {
		return nil, err
	}
	fileSystem.InodeTable = make(map[uint64]*Inode)
	fileSystem.inode, _ = fileSystem.getInode(&stat)
	err = fileSystem.scan(&fileSystem, "")
	if err != nil {
		return nil, err
	}
	if cacheDirectoryName != "" {
		fileSystem.ObjectCache = make([][]byte, 0, 65536)
		fileSystem.ObjectCache, err = scanObjectCache(cacheDirectoryName, "",
			fileSystem.ObjectCache)
		if err != nil {
			return nil, err
		}
	}
	return &fileSystem, nil
}
