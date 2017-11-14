package scanner

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path"
	"syscall"

	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/log"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
	filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP |
		syscall.S_IROTH
)

func (imdb *ImageDataBase) addImage(image *image.Image, name string,
	username *string) error {
	if err := image.Verify(); err != nil {
		return err
	}
	if imageIsExpired(image) {
		imdb.logger.Printf("Ignoring already expired image: %s\n", name)
		return nil
	}
	imdb.Lock()
	defer imdb.Unlock()
	if _, ok := imdb.imageMap[name]; ok {
		return errors.New("image: " + name + " already exists")
	} else {
		err := imdb.checkDirectoryPermissions(path.Dir(name), username)
		if err != nil {
			return err
		}
		filename := path.Join(imdb.baseDir, name)
		flags := os.O_CREATE | os.O_RDWR
		if imdb.masterMode {
			flags |= os.O_EXCL
		} else {
			flags |= os.O_TRUNC
		}
		file, err := os.OpenFile(filename, flags, filePerms)
		if err != nil {
			if os.IsExist(err) {
				return errors.New("cannot add previously deleted image: " +
					name)
			}
			return err
		}
		defer file.Close()
		w := bufio.NewWriter(file)
		defer w.Flush()
		writer := fsutil.NewChecksumWriter(w)
		defer writer.WriteChecksum()
		encoder := gob.NewEncoder(writer)
		if err := encoder.Encode(image); err != nil {
			os.Remove(filename)
			return err
		}
		if err := w.Flush(); err != nil {
			os.Remove(filename)
			return err
		}
		imdb.scheduleExpiration(image, name)
		imdb.imageMap[name] = image
		imdb.addNotifiers.sendPlain(name, "add", imdb.logger)
		imdb.removeFromUnreferencedObjectsListAndSave(
			image.FileSystem.InodeTable)
		return nil
	}
}

// This must be called with the lock held.
func (imdb *ImageDataBase) checkDirectoryPermissions(dirname string,
	username *string) error {
	if username == nil {
		return nil
	}
	directoryMetadata, ok := imdb.directoryMap[dirname]
	if !ok {
		return fmt.Errorf("no metadata for: \"%s\"", dirname)
	}
	if directoryMetadata.OwnerGroup == "" {
		return nil
	}
	if *username == "" {
		return errors.New("no username: unauthenticated connection")
	}
	return checkUserInGroup(*username, directoryMetadata.OwnerGroup)
}

func (imdb *ImageDataBase) checkImage(name string) bool {
	imdb.RLock()
	defer imdb.RUnlock()
	_, ok := imdb.imageMap[name]
	return ok
}

func (imdb *ImageDataBase) chownDirectory(dirname, ownerGroup string) error {
	dirname = path.Clean(dirname)
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
	oldDirectoryMetadata, ok := imdb.directoryMap[directory.Name]
	if ok && directory.Metadata == oldDirectoryMetadata {
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
	_, ok := imdb.directoryMap[directory.Name]
	if directory.Metadata == (image.DirectoryMetadata{}) {
		if !ok {
			return nil
		}
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

func (imdb *ImageDataBase) countDirectories() uint {
	imdb.RLock()
	defer imdb.RUnlock()
	return uint(len(imdb.directoryMap))
}

func (imdb *ImageDataBase) countImages() uint {
	imdb.RLock()
	defer imdb.RUnlock()
	return uint(len(imdb.imageMap))
}

func (imdb *ImageDataBase) deleteImage(name string, username *string) error {
	imdb.Lock()
	defer imdb.Unlock()
	if _, ok := imdb.imageMap[name]; ok {
		err := imdb.checkDirectoryPermissions(path.Dir(name), username)
		if err != nil {
			return err
		}
		filename := path.Join(imdb.baseDir, name)
		if err := os.Truncate(filename, 0); err != nil {
			return err
		}
		imdb.deleteImageAndUpdateUnreferencedObjectsList(name)
		imdb.deleteNotifiers.sendPlain(name, "delete", imdb.logger)
		return nil
	} else {
		return errors.New("image: " + name + " does not exist")
	}
}

func (imdb *ImageDataBase) deleteImageAndUpdateUnreferencedObjectsList(
	name string) {
	img := imdb.imageMap[name]
	if img == nil { // May be nil if expiring an already deleted image.
		return
	}
	delete(imdb.imageMap, name)
	imdb.maybeAddToUnreferencedObjectsList(img.FileSystem)
}

func (imdb *ImageDataBase) deleteUnreferencedObjects(percentage uint8,
	bytesThreshold uint64) error {
	objects := imdb.listUnreferencedObjects()
	objectsThreshold := uint64(percentage) * uint64(len(objects)) / 100
	var objectsCount, bytesCount uint64
	for hashVal, size := range objects {
		if !(objectsCount < objectsThreshold || bytesCount < bytesThreshold) {
			break
		}
		if err := imdb.objectServer.DeleteObject(hashVal); err != nil {
			imdb.logger.Printf("Error cleaning up: %x: %s\n", hashVal, err)
			return fmt.Errorf("error cleaning up: %x: %s\n", hashVal, err)
		}
		objectsCount++
		bytesCount += size
	}
	return nil
}

func (imdb *ImageDataBase) doWithPendingImage(image *image.Image,
	doFunc func() error) error {
	imdb.pendingImageLock.Lock()
	defer imdb.pendingImageLock.Unlock()
	imdb.Lock()
	changed := imdb.removeFromUnreferencedObjectsList(
		image.FileSystem.InodeTable)
	imdb.Unlock()
	err := doFunc()
	imdb.Lock()
	defer imdb.Unlock()
	for _, img := range imdb.imageMap {
		if img == image { // image was added, save if change happened above.
			if changed {
				imdb.saveUnreferencedObjectsList(false)
			}
			return err
		}
	}
	// image was not added: "delete" it by maybe adding to unreferenced list.
	imdb.maybeAddToUnreferencedObjectsList(image.FileSystem)
	return err
}

func (imdb *ImageDataBase) getImage(name string) *image.Image {
	imdb.RLock()
	defer imdb.RUnlock()
	return imdb.imageMap[name]
}

func (imdb *ImageDataBase) getUnreferencedObjectsStatistics() (uint64, uint64) {
	imdb.maybeRegenerateUnreferencedObjectsList()
	imdb.RLock()
	defer imdb.RUnlock()
	return uint64(len(imdb.unreferencedObjects.hashToEntry)),
		imdb.unreferencedObjects.totalBytes
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

func (imdb *ImageDataBase) listUnreferencedObjects() map[hash.Hash]uint64 {
	imdb.maybeRegenerateUnreferencedObjectsList()
	imdb.RLock()
	defer imdb.RUnlock()
	return imdb.unreferencedObjects.list()
}

func (imdb *ImageDataBase) makeDirectory(directory image.Directory,
	username string, userRpc bool) error {
	directory.Name = path.Clean(directory.Name)
	pathname := path.Join(imdb.baseDir, directory.Name)
	imdb.Lock()
	defer imdb.Unlock()
	oldDirectoryMetadata, ok := imdb.directoryMap[directory.Name]
	if userRpc {
		if ok {
			return fmt.Errorf("directory: %s already exists", directory.Name)
		}
		directory.Metadata = oldDirectoryMetadata
		parentMetadata, ok := imdb.directoryMap[path.Dir(directory.Name)]
		if !ok {
			return fmt.Errorf("no metadata for: %s", path.Dir(directory.Name))
		}
		if parentMetadata.OwnerGroup != "" {
			if err := checkUserInGroup(username,
				parentMetadata.OwnerGroup); err != nil {
				return err
			}
		}
		directory.Metadata.OwnerGroup = parentMetadata.OwnerGroup
	}
	if err := os.Mkdir(pathname, dirPerms); err != nil && !os.IsExist(err) {
		return err
	}
	return imdb.updateDirectoryMetadata(directory)
}

func checkUserInGroup(username, ownerGroup string) error {
	userData, err := user.Lookup(username)
	if err != nil {
		return err
	}
	groupData, err := user.LookupGroup(ownerGroup)
	if err != nil {
		return err
	}
	groupIDs, err := userData.GroupIds()
	if err != nil {
		return err
	}
	for _, groupID := range groupIDs {
		if groupData.Gid == groupID {
			return nil
		}
	}
	return fmt.Errorf("user: %s not a member of: %s", username, ownerGroup)
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
	logger log.Logger) {
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
	logger log.Logger) {
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
