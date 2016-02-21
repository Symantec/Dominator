package scanner

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/objectserver"
	"log"
	"os"
	"path"
	"syscall"
	"time"
)

func loadImageDataBase(baseDir string, objSrv objectserver.ObjectServer,
	logger *log.Logger) (*ImageDataBase, error) {
	fi, err := os.Stat(baseDir)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Cannot stat: %s\t%s\n", baseDir, err))
	}
	if !fi.IsDir() {
		return nil, errors.New(fmt.Sprintf("%s is not a directory\n", baseDir))
	}
	imdb := new(ImageDataBase)
	imdb.baseDir = baseDir
	imdb.imageMap = make(map[string]*image.Image)
	imdb.addNotifiers = make(notifiers)
	imdb.deleteNotifiers = make(notifiers)
	imdb.objectServer = objSrv
	imdb.logger = logger
	startTime := time.Now()
	var rusageStart, rusageStop syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	if err = imdb.scanDirectory(""); err != nil {
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
	return imdb, nil
}

func (imdb *ImageDataBase) scanDirectory(dirname string) error {
	file, err := os.Open(path.Join(imdb.baseDir, dirname))
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	for _, name := range names {
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
			err = imdb.scanDirectory(filename)
		} else if stat.Mode&syscall.S_IFMT == syscall.S_IFREG {
			err = imdb.loadFile(filename)
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

func (imdb *ImageDataBase) loadFile(filename string) error {
	file, err := os.Open(path.Join(imdb.baseDir, filename))
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	var image image.Image
	if err = decoder.Decode(&image); err != nil {
		return err
	}
	image.FileSystem.RebuildInodePointers()
	imdb.imageMap[filename] = &image
	return nil
}
