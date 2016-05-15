package scanner

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/image"
	"io"
	"log"
	"os"
	"path"
	"syscall"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
	filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP |
		syscall.S_IROTH
)

func (imdb *ImageDataBase) addImage(image *image.Image, name string) error {
	if err := image.Verify(); err != nil {
		return err
	}
	imdb.Lock()
	defer imdb.Unlock()
	if _, ok := imdb.imageMap[name]; ok {
		return errors.New("image: " + name + " already exists")
	} else {
		filename := path.Join(imdb.baseDir, name)
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_RDWR,
			filePerms)
		if err != nil {
			if os.IsExist(err) {
				return errors.New("cannot add a previously deleted image")
			}
			return err
		}
		defer file.Close()
		w := bufio.NewWriter(file)
		defer w.Flush()
		writer := fsutil.NewChecksumWriter(w)
		defer writer.WriteChecksum()
		encoder := gob.NewEncoder(writer)
		encoder.Encode(image)
		imdb.imageMap[name] = image
		imdb.addNotifiers.sendPlain(name, "add", imdb.logger)
		return nil
	}
}

func (imdb *ImageDataBase) checkImage(name string) bool {
	imdb.RLock()
	defer imdb.RUnlock()
	_, ok := imdb.imageMap[name]
	return ok
}

func (imdb *ImageDataBase) chownDirectory(dirname, ownerGroup string) error {
	imdb.RLock()
	directoryMetadata, ok := imdb.directoryMap[dirname]
	imdb.RUnlock()
	if !ok {
		return fmt.Errorf("no metadata for: \"%s\"", dirname)
	}
	directoryMetadata.OwnerGroup = ownerGroup
	imdb.Lock()
	defer imdb.Unlock()
	return imdb.updateDirectoryMetadata(
		image.Directory{Name: dirname, Metadata: directoryMetadata})
}

// This must be called with the lock held.
func (imdb *ImageDataBase) updateDirectoryMetadata(
	directory image.Directory) error {
	if directory.Metadata == imdb.directoryMap[directory.Name] {
		return nil
	}
	if err := imdb.updateDirectoryMetadataFile(directory); err != nil {
		return err
	}
	imdb.directoryMap[directory.Name] = directory.Metadata
	imdb.mkdirNotifiers.sendMakeDirectory(directory, imdb.logger)
	return nil
}

func (imdb *ImageDataBase) updateDirectoryMetadataFile(
	directory image.Directory) error {
	filename := path.Join(imdb.baseDir, directory.Name, metadataFile)
	if directory.Metadata == (image.DirectoryMetadata{}) {
		return os.Remove(filename)
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, filePerms)
	if err != nil {
		return err
	}
	if err := writeDirectoryMetadata(file, directory.Metadata); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func writeDirectoryMetadata(file io.Writer,
	directoryMetadata image.DirectoryMetadata) error {
	w := bufio.NewWriter(file)
	writer := fsutil.NewChecksumWriter(w)
	if err := gob.NewEncoder(writer).Encode(directoryMetadata); err != nil {
		return err
	}
	if err := writer.WriteChecksum(); err != nil {
		return err
	}
	return w.Flush()
}

func (imdb *ImageDataBase) countImages() uint {
	imdb.RLock()
	defer imdb.RUnlock()
	return uint(len(imdb.imageMap))
}

func (imdb *ImageDataBase) deleteImage(name string) error {
	imdb.Lock()
	defer imdb.Unlock()
	if _, ok := imdb.imageMap[name]; ok {
		filename := path.Join(imdb.baseDir, name)
		if err := os.Truncate(filename, 0); err != nil {
			return err
		}
		delete(imdb.imageMap, name)
		imdb.deleteNotifiers.sendPlain(name, "delete", imdb.logger)
		return nil
	} else {
		return errors.New("image: " + name + " does not exist")
	}
}

func (imdb *ImageDataBase) getImage(name string) *image.Image {
	imdb.RLock()
	defer imdb.RUnlock()
	return imdb.imageMap[name]
}

func (imdb *ImageDataBase) listDirectories() []image.Directory {
	imdb.RLock()
	defer imdb.RUnlock()
	directories := make([]image.Directory, 0, len(imdb.directoryMap))
	for name, metadata := range imdb.directoryMap {
		directories = append(directories,
			image.Directory{Name: name, Metadata: metadata})
	}
	return directories
}

func (imdb *ImageDataBase) listImages() []string {
	imdb.RLock()
	defer imdb.RUnlock()
	names := make([]string, 0)
	for name := range imdb.imageMap {
		names = append(names, name)
	}
	return names
}

func (imdb *ImageDataBase) makeDirectory(directory image.Directory,
	username string, userRpc bool) error {
	pathname := path.Join(imdb.baseDir, directory.Name)
	imdb.Lock()
	defer imdb.Unlock()
	directoryMetadata := imdb.directoryMap[directory.Name]
	if err := os.Mkdir(pathname, dirPerms); err != nil && !os.IsExist(err) {
		return err
	}
	if userRpc {
		directory.Metadata = directoryMetadata
		if directory.Metadata.OwnerGroup != "" {
			// TODO(rgooch): Check if username is part of owner group.
		}
	}
	return imdb.updateDirectoryMetadata(directory)
}

func (imdb *ImageDataBase) registerAddNotifier() <-chan string {
	channel := make(chan string, 1)
	imdb.Lock()
	defer imdb.Unlock()
	imdb.addNotifiers[channel] = channel
	return channel
}

func (imdb *ImageDataBase) registerDeleteNotifier() <-chan string {
	channel := make(chan string, 1)
	imdb.Lock()
	defer imdb.Unlock()
	imdb.deleteNotifiers[channel] = channel
	return channel
}

func (imdb *ImageDataBase) registerMakeDirectoryNotifier() <-chan image.Directory {
	channel := make(chan image.Directory, 1)
	imdb.Lock()
	defer imdb.Unlock()
	imdb.mkdirNotifiers[channel] = channel
	return channel
}

func (imdb *ImageDataBase) unregisterAddNotifier(channel <-chan string) {
	imdb.Lock()
	defer imdb.Unlock()
	delete(imdb.addNotifiers, channel)
}

func (imdb *ImageDataBase) unregisterDeleteNotifier(channel <-chan string) {
	imdb.Lock()
	defer imdb.Unlock()
	delete(imdb.deleteNotifiers, channel)
}

func (imdb *ImageDataBase) unregisterMakeDirectoryNotifier(
	channel <-chan image.Directory) {
	imdb.Lock()
	defer imdb.Unlock()
	delete(imdb.mkdirNotifiers, channel)
}

func (n notifiers) sendPlain(name string, operation string,
	logger *log.Logger) {
	if len(n) < 1 {
		return
	} else {
		plural := "s"
		if len(n) < 2 {
			plural = ""
		}
		logger.Printf("Sending %s notification to: %d listener%s\n",
			operation, len(n), plural)
	}
	for _, sendChannel := range n {
		go func(channel chan<- string) {
			channel <- name
		}(sendChannel)
	}
}

func (n makeDirectoryNotifiers) sendMakeDirectory(dir image.Directory,
	logger *log.Logger) {
	if len(n) < 1 {
		return
	} else {
		plural := "s"
		if len(n) < 2 {
			plural = ""
		}
		logger.Printf("Sending mkdir notification to: %d listener%s\n",
			len(n), plural)
	}
	for _, sendChannel := range n {
		go func(channel chan<- image.Directory) {
			channel <- dir
		}(sendChannel)
	}
}
