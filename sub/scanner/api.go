package scanner

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsrateio"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/rateio"
	"io"
	"log"
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

type FileSystemHistory struct {
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
	return fsh.fileSystem
}

func (fsh *FileSystemHistory) GenerationCount() uint64 {
	return fsh.generationCount
}

func (fsh FileSystemHistory) String() string {
	return fmt.Sprintf("GenerationCount=%d\n", fsh.generationCount)
}

func (fsh *FileSystemHistory) WriteHtml(writer io.Writer) {
	fsh.writeHtml(writer)
}

type FileSystem struct {
	configuration     *Configuration
	rootDirectoryName string
	dev               uint64
	inodeNumber       uint64
	filesystem.FileSystem
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
	configuration *Configuration, logger *log.Logger) chan *FileSystem {
	return startScannerDaemon(rootDirectoryName, cacheDirectoryName,
		configuration, logger)
}
