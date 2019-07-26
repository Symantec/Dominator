package scanner

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
	filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP |
		syscall.S_IROTH
)

var (
	errNoAccess   = errors.New("no access to image")
	errNoAuthInfo = errors.New("no authentication information")
)

func (imdb *ImageDataBase) addImage(image *image.Image, name string,
	authInfo *srpc.AuthInformation) error {
	if err := image.Verify(); err != nil {
		return err
	}
	if imageIsExpired(image) {
		imdb.logger.Printf("Ignoring already expired image: %s\n", name)
		return nil
	}
	imdb.deduperLock.Lock()
	image.ReplaceStrings(imdb.deduper.DeDuplicate)
	imdb.deduperLock.Unlock()
	imdb.Lock()
	defer imdb.Unlock()
	if _, ok := imdb.imageMap[name]; ok {
		return errors.New("image: " + name + " already exists")
	} else {
		if err := imdb.checkPermissions(name, authInfo); err != nil {
			return err
		}
		filename := filepath.Join(imdb.baseDir, name)
		flags := os.O_CREATE | os.O_RDWR
		if imdb.replicationMaster != "" {
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
		imdb.removeFromUnreferencedObjectsListAndSave(image)
		return nil
	}
}

func (imdb *ImageDataBase) changeImageExpiration(name string,
	expiresAt time.Time, authInfo *srpc.AuthInformation) (bool, error) {
	imdb.Lock()
	defer imdb.Unlock()
	if img, ok := imdb.imageMap[name]; !ok {
		return false, errors.New("image not found")
	} else if err := imdb.checkPermissions(name, authInfo); err != nil {
		return false, err
	} else if img.ExpiresAt.IsZero() {
		return false, errors.New("image does not expire")
	} else if expiresAt.IsZero() {
		if err := imdb.writeNewExpiration(name, img, expiresAt); err != nil {
			return false, err
		}
		img.ExpiresAt = expiresAt
		imdb.addNotifiers.sendPlain(name, "add", imdb.logger)
		return true, nil
	} else if expiresAt.Before(img.ExpiresAt) {
		return false, errors.New("cannot shorten expiration time")
	} else if expiresAt.After(img.ExpiresAt) {
		if err := imdb.writeNewExpiration(name, img, expiresAt); err != nil {
			return false, err
		}
		img.ExpiresAt = expiresAt
		imdb.addNotifiers.sendPlain(name, "add", imdb.logger)
		return true, nil
	} else {
		return false, nil
	}
}

// This must be called with the lock held.
func (imdb *ImageDataBase) checkChown(dirname, ownerGroup string,
	authInfo *srpc.AuthInformation) error {
	if authInfo == nil {
		return errNoAuthInfo
	}
	if authInfo.HaveMethodAccess {
		return nil
	}
	// If owner of parent, any group can be set.
	parentDirname := filepath.Dir(dirname)
	if directoryMetadata, ok := imdb.directoryMap[parentDirname]; ok {
		if directoryMetadata.OwnerGroup != "" {
			if _, ok := authInfo.GroupList[directoryMetadata.OwnerGroup]; ok {
				return nil
			}
		}
	}
	if _, ok := authInfo.GroupList[ownerGroup]; !ok {
		return fmt.Errorf("no membership of %s group", ownerGroup)
	}
	if directoryMetadata, ok := imdb.directoryMap[dirname]; !ok {
		return fmt.Errorf("no metadata for: \"%s\"", dirname)
	} else if directoryMetadata.OwnerGroup != "" {
		if _, ok := authInfo.GroupList[directoryMetadata.OwnerGroup]; ok {
			return nil
		}
	}
	return errNoAccess
}

func (imdb *ImageDataBase) checkDirectory(name string) bool {
	imdb.RLock()
	defer imdb.RUnlock()
	_, ok := imdb.directoryMap[name]
	return ok
}

func (imdb *ImageDataBase) checkImage(name string) bool {
	imdb.RLock()
	defer imdb.RUnlock()
	_, ok := imdb.imageMap[name]
	return ok
}

// This must be called with the lock held.
func (imdb *ImageDataBase) checkPermissions(imageName string,
	authInfo *srpc.AuthInformation) error {
	if authInfo == nil {
		return errNoAuthInfo
	}
	if authInfo.HaveMethodAccess {
		return nil
	}
	dirname := filepath.Dir(imageName)
	if directoryMetadata, ok := imdb.directoryMap[dirname]; !ok {
		return fmt.Errorf("no metadata for: \"%s\"", dirname)
	} else if directoryMetadata.OwnerGroup != "" {
		if _, ok := authInfo.GroupList[directoryMetadata.OwnerGroup]; ok {
			return nil
		}
	}
	return errNoAccess
}

func (imdb *ImageDataBase) chownDirectory(dirname, ownerGroup string,
	authInfo *srpc.AuthInformation) error {
	dirname = filepath.Clean(dirname)
	imdb.Lock()
	defer imdb.Unlock()
	directoryMetadata, ok := imdb.directoryMap[dirname]
	if !ok {
		return fmt.Errorf("no metadata for: \"%s\"", dirname)
	}
	if err := imdb.checkChown(dirname, ownerGroup, authInfo); err != nil {
		return err
	}
	directoryMetadata.OwnerGroup = ownerGroup
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
	filename := filepath.Join(imdb.baseDir, directory.Name, metadataFile)
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

func (imdb *ImageDataBase) deleteImage(name string,
	authInfo *srpc.AuthInformation) error {
	imdb.Lock()
	defer imdb.Unlock()
	if _, ok := imdb.imageMap[name]; ok {
		if err := imdb.checkPermissions(name, authInfo); err != nil {
			return err
		}
		filename := filepath.Join(imdb.baseDir, name)
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
	imdb.rebuildDeDuper()
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
	changed := imdb.removeFromUnreferencedObjectsList(image)
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

func (imdb *ImageDataBase) findLatestImage(dirname string,
	ignoreExpiring bool) (string, error) {
	imdb.RLock()
	defer imdb.RUnlock()
	if _, ok := imdb.directoryMap[dirname]; !ok {
		return "", errors.New("unknown directory: " + dirname)
	}
	var previousCreateTime time.Time
	var imageName string
	for name, img := range imdb.imageMap {
		if ignoreExpiring && !img.ExpiresAt.IsZero() {
			continue
		}
		if filepath.Dir(name) != dirname {
			continue
		}
		if img.CreatedOn.After(previousCreateTime) {
			imageName = name
			previousCreateTime = img.CreatedOn
		}
	}
	return imageName, nil
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
	authInfo *srpc.AuthInformation, userRpc bool) error {
	directory.Name = filepath.Clean(directory.Name)
	pathname := filepath.Join(imdb.baseDir, directory.Name)
	imdb.Lock()
	defer imdb.Unlock()
	oldDirectoryMetadata, ok := imdb.directoryMap[directory.Name]
	if userRpc {
		if authInfo == nil {
			return errNoAuthInfo
		}
		if ok {
			return fmt.Errorf("directory: %s already exists", directory.Name)
		}
		directory.Metadata = oldDirectoryMetadata
		parentMetadata, ok := imdb.directoryMap[filepath.Dir(directory.Name)]
		if !ok {
			return fmt.Errorf("no metadata for: %s",
				filepath.Dir(directory.Name))
		}
		if !authInfo.HaveMethodAccess {
			if parentMetadata.OwnerGroup == "" {
				return errNoAccess
			}
			if _, ok := authInfo.GroupList[parentMetadata.OwnerGroup]; !ok {
				return fmt.Errorf("no membership of %s group",
					parentMetadata.OwnerGroup)
			}
		}
		directory.Metadata.OwnerGroup = parentMetadata.OwnerGroup
	}
	if err := os.Mkdir(pathname, dirPerms); err != nil && !os.IsExist(err) {
		return err
	}
	return imdb.updateDirectoryMetadata(directory)
}

// This must be called with the main lock held.
func (imdb *ImageDataBase) rebuildDeDuper() {
	imdb.deduperLock.Lock()
	defer imdb.deduperLock.Unlock()
	startTime := time.Now()
	imdb.deduper.Clear()
	for _, image := range imdb.imageMap {
		image.ReplaceStrings(imdb.deduper.DeDuplicate)
	}
	imdb.logger.Debugf(0, "Rebuilding de-duper state took %s\n",
		time.Since(startTime))
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

// This must be called with the lock held.
func (imdb *ImageDataBase) writeNewExpiration(name string,
	oldImage *image.Image, expiresAt time.Time) error {
	img := *oldImage
	img.ExpiresAt = expiresAt
	filename := filepath.Join(imdb.baseDir, name)
	tmpFilename := filename + "~"
	file, err := os.OpenFile(tmpFilename, os.O_CREATE|os.O_RDWR|os.O_EXCL,
		filePerms)
	if err != nil {
		return err
	}
	defer file.Close()
	defer os.Remove(tmpFilename)
	w := bufio.NewWriter(file)
	defer w.Flush()
	writer := fsutil.NewChecksumWriter(w)
	encoder := gob.NewEncoder(writer)
	if err := encoder.Encode(img); err != nil {
		return err
	}
	if err := writer.WriteChecksum(); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fsutil.FsyncFile(file)
	return os.Rename(tmpFilename, filename)
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
