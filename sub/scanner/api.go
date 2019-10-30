package scanner

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/cpulimiter"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem/scanner"
	"github.com/Cloud-Foundations/Dominator/lib/filter"
	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/fsrateio"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
	"github.com/Cloud-Foundations/Dominator/lib/rateio"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
)

type Configuration struct {
	CpuLimiter           *cpulimiter.CpuLimiter
	DefaultCpuPercent    uint
	FsScanContext        *fsrateio.ReaderContext
	NetworkReaderContext *rateio.ReaderContext
	ScanFilter           *filter.Filter
}

func (configuration *Configuration) BoostCpuLimit(logger log.Logger) {
	configuration.boostCpuLimit(logger)
}

func (configuration *Configuration) RegisterMetrics(
	dir *tricorder.DirectorySpec) error {
	return configuration.registerMetrics(dir)
}

func (configuration *Configuration) RestoreCpuLimit(logger log.Logger) {
	configuration.restoreCpuLimit(logger)
}

func (configuration *Configuration) WriteHtml(writer io.Writer) {
	configuration.writeHtml(writer)
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

func (fsh *FileSystemHistory) DurationOfLastScan() time.Duration {
	fsh.rwMutex.RLock()
	defer fsh.rwMutex.RUnlock()
	return fsh.durationOfLastScan
}

func (fsh *FileSystemHistory) FileSystem() *FileSystem {
	fsh.rwMutex.RLock()
	defer fsh.rwMutex.RUnlock()
	return fsh.fileSystem
}

func (fsh *FileSystemHistory) GenerationCount() uint64 {
	fsh.rwMutex.RLock()
	defer fsh.rwMutex.RUnlock()
	return fsh.generationCount
}

func (fsh *FileSystemHistory) ScanCount() uint64 {
	fsh.rwMutex.RLock()
	defer fsh.rwMutex.RUnlock()
	return fsh.scanCount
}

func (fsh *FileSystemHistory) String() string {
	fsh.rwMutex.RLock()
	defer fsh.rwMutex.RUnlock()
	return fmt.Sprintf("GenerationCount=%d\n", fsh.generationCount)
}

func (fsh *FileSystemHistory) Update(newFS *FileSystem) {
	fsh.update(newFS)
}

func (fsh *FileSystemHistory) UpdateObjectCacheOnly() error {
	return fsh.updateObjectCacheOnly()
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
		&FileSystem{})
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
	configuration *Configuration, logger log.Logger) (
	<-chan *FileSystem, func(disableScanner bool)) {
	return startScannerDaemon(rootDirectoryName, cacheDirectoryName,
		configuration, logger)
}

func StartScanning(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration, logger log.Logger,
	mainFunc func(<-chan *FileSystem, func(disableScanner bool))) {
	startScanning(rootDirectoryName, cacheDirectoryName, configuration, logger,
		mainFunc)
}
