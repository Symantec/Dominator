package scanner

import (
	"github.com/Symantec/Dominator/lib/filesystem/scanner"
	"github.com/Symantec/Dominator/lib/objectcache"
)

func scanFileSystem(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration, oldFS *FileSystem) (*FileSystem, error) {
	var fileSystem FileSystem
	fileSystem.configuration = configuration
	fileSystem.rootDirectoryName = rootDirectoryName
	fileSystem.cacheDirectoryName = cacheDirectoryName
	fs, err := scanner.ScanFileSystem(rootDirectoryName,
		configuration.FsScanContext, configuration.ScanFilter,
		checkScanDisableRequest, &oldFS.FileSystem)
	if err != nil {
		return nil, err
	}
	fileSystem.FileSystem = *fs
	if err = fileSystem.scanObjectCache(); err != nil {
		return nil, err
	}
	return &fileSystem, nil
}

func (fs *FileSystem) scanObjectCache() error {
	if fs.cacheDirectoryName == "" {
		return nil
	}
	var err error
	fs.ObjectCache, err = objectcache.ScanObjectCache(fs.cacheDirectoryName)
	return err
}
