package scanner

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem/scanner"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsrateio"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/rateio"
	"github.com/Symantec/tricorder/go/tricorder"
	"io"
	"log"
	"sync"
	"time"
)

type Configuration struct {
	FsScanContext        *fsrateio.ReaderContext
	NetworkReaderContext *rateio.ReaderContext
	ScanFilter           *filter.Filter
}

func (configuration *Configuration) WriteHtml(writer io.Writer) {
	configuration.writeHtml(writer)
}

func (configuration *Configuration) RegisterMetrics(
	dir *tricorder.DirectorySpec) error {
	return configuration.registerMetrics(dir)
}

type FileSystemHistory struct {
	rwMutex            sync.RWMutex
	fileSystem         *FileSystem
	scanCount          uint64
	generationCount    uint64
	timeOfLastScan     time.Time
	durationOfLastScan time.Duration
	timeOfLastChange   time.Time
}

func (fsh *FileSystemHistory) Update(newFS *FileSystem) {
	fsh.update(newFS)
}

func (fsh *FileSystemHistory) UpdateObjectCacheOnly() error {
	return fsh.updateObjectCacheOnly()
}

func (fsh *FileSystemHistory) FileSystem() *FileSystem {
	fsh.rwMutex.RLock()
	defer fsh.rwMutex.RUnlock()
	return fsh.fileSystem
}

func (fsh *FileSystemHistory) ScanCount() uint64 {
	fsh.rwMutex.RLock()
	defer fsh.rwMutex.RUnlock()
	return fsh.scanCount
}

func (fsh *FileSystemHistory) GenerationCount() uint64 {
	fsh.rwMutex.RLock()
	defer fsh.rwMutex.RUnlock()
	return fsh.generationCount
}

func (fsh FileSystemHistory) String() string {
	fsh.rwMutex.RLock()
	defer fsh.rwMutex.RUnlock()
	return fmt.Sprintf("GenerationCount=%d\n", fsh.generationCount)
}

func (fsh *FileSystemHistory) WriteHtml(writer io.Writer) {
	fsh.rwMutex.RLock()
	defer fsh.rwMutex.RUnlock()
	fsh.writeHtml(writer)
}

type FileSystem struct {
	configuration     *Configuration
	rootDirectoryName string
	scanner.FileSystem
	cacheDirectoryName string
	objectcache.ObjectCache
}

func ScanFileSystem(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration) (*FileSystem, error) {
	return scanFileSystem(rootDirectoryName, cacheDirectoryName, configuration,
		nil)
}

func (fs *FileSystem) ScanObjectCache() error {
	return fs.scanObjectCache()
}

func (fs *FileSystem) Configuration() *Configuration {
	return fs.configuration
}

func (fs *FileSystem) RootDirectoryName() string {
	return fs.rootDirectoryName
}

func (fs *FileSystem) String() string {
	return fmt.Sprintf("Tree: %d inodes, total file size: %s, number of regular inodes: %d\nObjectCache: %d objects\n",
		len(fs.InodeTable),
		format.FormatBytes(fs.TotalDataBytes),
		fs.NumRegularInodes,
		len(fs.ObjectCache))
}

func (fs *FileSystem) WriteHtml(writer io.Writer) {
	fs.writeHtml(writer)
}

func CompareFileSystems(left, right *FileSystem, logWriter io.Writer) bool {
	return compareFileSystems(left, right, logWriter)
}

func StartScannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration, logger *log.Logger) (
	<-chan *FileSystem, func(disableScanner bool)) {
	return startScannerDaemon(rootDirectoryName, cacheDirectoryName,
		configuration, logger)
}
