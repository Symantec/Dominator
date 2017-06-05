package scanner

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/concurrent"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/objectserver"
	"io"
	"log"
	"os"
	"path"
	"syscall"
	"time"
)

func loadImageDataBase(baseDir string, objSrv objectserver.FullObjectServer,
	masterMode bool, logger *log.Logger) (*ImageDataBase, error) {
	fi, err := os.Stat(baseDir)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Cannot stat: %s: %s\n", baseDir, err))
	}
	if !fi.IsDir() {
		return nil, errors.New(fmt.Sprintf("%s is not a directory\n", baseDir))
	}
	imdb := &ImageDataBase{
		baseDir:         baseDir,
		directoryMap:    make(map[string]image.DirectoryMetadata),
		imageMap:        make(map[string]*image.Image),
		addNotifiers:    make(notifiers),
		deleteNotifiers: make(notifiers),
		mkdirNotifiers:  make(makeDirectoryNotifiers),
		objectServer:    objSrv,
		masterMode:      masterMode,
		logger:          logger,
	}
	state := concurrent.NewState(0)
	startTime := time.Now()
	var rusageStart, rusageStop syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	if err := imdb.scanDirectory(".", state, logger); err != nil {
		return nil, err
	}
	if err := state.Reap(); err != nil {
		return nil, err
	}
	if logger != nil {
		plural := ""
		if imdb.CountImages() != 1 {
			plural = "s"
		}
		syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStop)
		userTime := time.Duration(rusageStop.Utime.Sec)*time.Second +
			time.Duration(rusageStop.Utime.Usec)*time.Microsecond -
			time.Duration(rusageStart.Utime.Sec)*time.Second -
			time.Duration(rusageStart.Utime.Usec)*time.Microsecond
		logger.Printf("Loaded %d image%s in %s (%s user CPUtime)\n",
			imdb.CountImages(), plural, time.Since(startTime), userTime)
	}
	imdb.unreferencedObjects, err = loadUnreferencedObjects(
		path.Join(baseDir, unreferencedObjectsFile))
	if err != nil {
		return nil, errors.New("error loading unreferenced objects list: " +
			err.Error())
	}
	imdb.regenerateUnreferencedObjectsList()
	if gcs, ok := objSrv.(objectserver.GarbageCollectorSetter); ok {
		gcs.SetGarbageCollector(imdb.garbageCollector)
	}
	return imdb, nil
}

func (imdb *ImageDataBase) scanDirectory(dirname string,
	state *concurrent.State, logger *log.Logger) error {
	directoryMetadata, err := imdb.readDirectoryMetadata(dirname)
	if err != nil {
		return err
	}
	imdb.directoryMap[dirname] = directoryMetadata
	file, err := os.Open(path.Join(imdb.baseDir, dirname))
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		if len(name) > 0 && name[0] == '.' {
			continue // Skip hidden paths.
		}
		filename := path.Join(dirname, name)
		var stat syscall.Stat_t
		err := syscall.Lstat(path.Join(imdb.baseDir, filename), &stat)
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return err
		}
		if stat.Mode&syscall.S_IFMT == syscall.S_IFDIR {
			err = imdb.scanDirectory(filename, state, logger)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFREG && stat.Size > 0 {
			err = state.GoRun(func() error {
				return imdb.loadFile(filename, logger)
			})
		}
		if err != nil {
			if err == syscall.ENOENT {
				continue
			}
			return err
		}
	}
	return nil
}

func (imdb *ImageDataBase) readDirectoryMetadata(dirname string) (
	image.DirectoryMetadata, error) {
	file, err := os.Open(path.Join(imdb.baseDir, dirname, metadataFile))
	if err != nil {
		if os.IsNotExist(err) {
			return image.DirectoryMetadata{}, nil
		}
		return image.DirectoryMetadata{}, err
	}
	defer file.Close()
	reader := fsutil.NewChecksumReader(file)
	decoder := gob.NewDecoder(reader)
	metadata := image.DirectoryMetadata{}
	if err := decoder.Decode(&metadata); err != nil {
		return image.DirectoryMetadata{}, fmt.Errorf(
			"unable to read directory metadata for \"%s\": %s", dirname, err)
	}
	return metadata, reader.VerifyChecksum()
}

func (imdb *ImageDataBase) loadFile(filename string, logger *log.Logger) error {
	pathname := path.Join(imdb.baseDir, filename)
	file, err := os.Open(pathname)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := fsutil.NewChecksumReader(file)
	decoder := gob.NewDecoder(reader)
	var image image.Image
	if err := decoder.Decode(&image); err != nil {
		return err
	}
	if err := reader.VerifyChecksum(); err != nil {
		if err == fsutil.ErrorChecksumMismatch {
			logger.Printf("Checksum mismatch for image: %s\n", filename)
			return nil
		}
		if err != io.EOF {
			return err
		}
	}
	image.FileSystem.RebuildInodePointers()
	if err := image.Verify(); err != nil {
		return err
	}
	imdb.Lock()
	defer imdb.Unlock()
	if imdb.scheduleExpiration(&image, filename) {
		imdb.logger.Printf("Deleting already expired image: %s\n", filename)
		return os.Remove(pathname)
	}
	imdb.imageMap[filename] = &image
	return nil
}
