package util

import (
	"io"
	"os"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/mbr"
	"github.com/Symantec/Dominator/lib/objectserver"
)

type ComputedFile struct {
	Filename string
	Source   string
}

type ComputedFilesData struct {
	FileData      map[string][]byte // Key: filename.
	RootDirectory string
}

// CopyMtimes will copy modification times for files from the source to the
// destination if the file data and metadata (other than mtime) are identical.
// Directory entry inode pointers are invalidated by this operation, so this
// should be followed by a call to dest.RebuildInodePointers().
func CopyMtimes(source, dest *filesystem.FileSystem) {
	copyMtimes(source, dest)
}

func GetUnsupportedExt4fsOptions(fs *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter) ([]string, error) {
	return getUnsupportedOptions(fs, objectsGetter)
}

func LoadComputedFiles(filename string) ([]ComputedFile, error) {
	return loadComputedFiles(filename)
}

func MakeBootable(fs *filesystem.FileSystem,
	deviceName, rootLabel, rootDir, kernelOptions string,
	doChroot bool, logger log.DebugLogger) error {
	return makeBootable(fs, deviceName, rootLabel, rootDir, kernelOptions,
		doChroot, logger)
}

func MakeExt4fs(deviceName, label string, unsupportedOptions []string,
	bytesPerInode uint64, logger log.Logger) error {
	return makeExt4fs(deviceName, label, unsupportedOptions, bytesPerInode,
		logger)
}

func ReplaceComputedFiles(fs *filesystem.FileSystem,
	computedFilesData *ComputedFilesData,
	objectsGetter objectserver.ObjectsGetter) (
	objectserver.ObjectsGetter, error) {
	return replaceComputedFiles(fs, computedFilesData, objectsGetter)
}

func SpliceComputedFiles(fs *filesystem.FileSystem,
	computedFileList []ComputedFile) error {
	return spliceComputedFiles(fs, computedFileList)
}

func Unpack(fs *filesystem.FileSystem, objectsGetter objectserver.ObjectsGetter,
	rootDir string, logger log.Logger) error {
	return unpack(fs, objectsGetter, rootDir, logger)
}

func WriteFstabEntry(writer io.Writer,
	source, mountPoint, fileSystemType, flags string,
	dumpFrequency, checkOrder uint) error {
	return writeFstabEntry(writer, source, mountPoint, fileSystemType, flags,
		dumpFrequency, checkOrder)
}

func WriteRaw(fs *filesystem.FileSystem,
	objectsGetter objectserver.ObjectsGetter, rawFilename string,
	perm os.FileMode, tableType mbr.TableType,
	minFreeSpace uint64, roundupPower uint64, makeBootable, allocateBlocks bool,
	logger log.DebugLogger) error {
	return writeRaw(fs, objectsGetter, rawFilename, perm, tableType,
		minFreeSpace, roundupPower, makeBootable, allocateBlocks, logger)
}
