package scanner

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsrateio"
	"github.com/Symantec/Dominator/lib/objectcache"
	"io"
	"regexp"
	"time"
)

type Configuration struct {
	FsScanContext *fsrateio.FsRateContext
	ExclusionList []*regexp.Regexp
}

func (configuration *Configuration) SetExclusionList(reList []string) error {
	return configuration.setExclusionList(reList)
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

type directoryInodeList map[uint64]bool

type FileSystem struct {
	configuration      *Configuration
	rootDirectoryName  string
	directoryInodeList directoryInodeList
	dev                uint64
	filesystem.FileSystem
	objectcache.ObjectCache
}

func ScanFileSystem(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration) (*FileSystem, error) {
	return scanFileSystem(rootDirectoryName, cacheDirectoryName, configuration,
		nil)
}

func (fs *FileSystem) Configuration() *Configuration {
	return fs.configuration
}

func (fs *FileSystem) String() string {
	return fmt.Sprintf("Tree: %d inodes, total file size: %s, number of hashes: %d\nObjectCache: %d objects\n",
		len(fs.RegularInodeTable)+len(fs.SymlinkInodeTable)+len(fs.InodeTable)+
			int(fs.DirectoryCount),
		format.FormatBytes(fs.TotalDataBytes),
		fs.HashCount,
		len(fs.ObjectCache))
}

func (fs *FileSystem) WriteHtml(writer io.Writer) {
	fs.writeHtml(writer)
}

func CompareFileSystems(left, right *FileSystem, logWriter io.Writer) bool {
	return compareFileSystems(left, right, logWriter)
}

func StartScannerDaemon(rootDirectoryName string, cacheDirectoryName string,
	configuration *Configuration) chan *FileSystem {
	return startScannerDaemon(rootDirectoryName, cacheDirectoryName,
		configuration)
}
