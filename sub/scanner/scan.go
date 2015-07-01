package scanner

import (
	"github.com/Symantec/Dominator/sub/fsrateio"
	"syscall"
)

func scanFileSystem(rootDirectoryName string, cacheDirectoryName string,
	ctx *fsrateio.FsRateContext) (*FileSystem, error) {
	var fileSystem FileSystem
	fileSystem.ctx = ctx
	fileSystem.Name = rootDirectoryName
	var stat syscall.Stat_t
	err := syscall.Lstat(rootDirectoryName, &stat)
	if err != nil {
		return nil, err
	}
	fileSystem.InodeTable = make(map[uint64]*Inode)
	fileSystem.Inode, _ = fileSystem.getInode(&stat)
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
	fileSystem.TotalDataBytes = fileSystem.computeTotalDataBytes()
	return &fileSystem, nil
}

func (fs *FileSystem) computeTotalDataBytes() uint64 {
	var totalBytes uint64 = 0
	for _, inode := range fs.InodeTable {
		totalBytes += uint64(inode.Stat.Size)
	}
	return totalBytes
}
