package unpacker

import (
	"errors"
	domlib "github.com/Symantec/Dominator/dom/lib"
	imageclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/objectcache"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	unpackproto "github.com/Symantec/Dominator/proto/imageunpacker"
	subproto "github.com/Symantec/Dominator/proto/sub"
	sublib "github.com/Symantec/Dominator/sub/lib"
	"io"
	"os"
	"path"
	"time"
)

func (u *Unpacker) unpackImage(streamName string, imageLeafName string) error {
	u.updateUsageTime()
	defer u.updateUsageTime()
	streamInfo := u.getStream(streamName)
	if streamInfo == nil {
		return errors.New("unknown stream")
	}
	fs := u.getImage(path.Join(streamName, imageLeafName)).FileSystem
	if err := fs.RebuildInodePointers(); err != nil {
		return err
	}
	fs.InodeToFilenamesTable()
	fs.FilenameToInodeTable()
	fs.HashToInodesTable()
	fs.ComputeTotalDataBytes()
	fs.BuildEntryMap()
	errorChannel := make(chan error)
	request := requestType{
		request:      requestUnpack,
		desiredFS:    fs,
		imageName:    path.Join(streamName, imageLeafName),
		errorChannel: errorChannel,
	}
	streamInfo.requestChannel <- request
	return <-errorChannel
}

func (u *Unpacker) getImage(imageName string) *image.Image {
	u.logger.Printf("Getting image: %s\n", imageName)
	interval := time.Second
	for ; true; time.Sleep(interval) {
		srpcClient, err := u.imageServerResource.GetHTTP(nil, 0)
		if err != nil {
			u.logger.Printf("Error connecting to image server: %s\n", err)
			continue
		}
		image, err := imageclient.GetImageWithTimeout(srpcClient, imageName,
			time.Minute)
		if err != nil {
			srpcClient.Close()
			u.logger.Printf("Error getting image: %s\n", err)
			continue
		}
		srpcClient.Put()
		if image != nil {
			return image
		}
		u.logger.Printf("Image: %s not ready yet\n", imageName)
		if interval < time.Second*10 {
			interval += time.Second
		}
	}
	return nil
}

func (stream *streamManagerState) unpack(imageName string,
	desiredFS *filesystem.FileSystem) error {
	mountPoint := path.Join(stream.unpacker.baseDir, "mnt")
	streamInfo := stream.streamInfo
	if streamInfo.status != unpackproto.StatusStreamScanned {
		return errors.New("not yet scanned")
	}
	err := stream.deleteUnneededFiles(imageName, stream.fileSystem, desiredFS,
		mountPoint)
	if err != nil {
		return err
	}
	subObj := domlib.Sub{
		FileSystem:  stream.fileSystem,
		ObjectCache: stream.objectCache,
	}
	stream.fileSystem = nil
	desiredImage := &image.Image{FileSystem: desiredFS}
	objectsToFetch, _ := domlib.BuildMissingLists(subObj, desiredImage, false,
		true, stream.unpacker.logger)
	objectsDir := path.Join(mountPoint, ".subd", "objects")
	if err := stream.fetch(imageName, objectsToFetch, objectsDir); err != nil {
		streamInfo.status = unpackproto.StatusStreamMounted
		return err
	}
	subObj.ObjectCache = append(subObj.ObjectCache, objectsToFetch...)
	streamInfo.status = unpackproto.StatusStreamUpdating
	stream.unpacker.logger.Printf("Update(%s) starting\n", imageName)
	startTime := time.Now()
	var request subproto.UpdateRequest
	domlib.BuildUpdateRequest(subObj, desiredImage, &request, true,
		stream.unpacker.logger)
	_, _, err = sublib.Update(request, mountPoint, objectsDir, nil, nil, nil,
		stream.unpacker.logger)
	streamInfo.status = unpackproto.StatusStreamMounted
	stream.unpacker.logger.Printf("Update(%s) completed in %s\n",
		imageName, format.Duration(time.Since(startTime)))
	return err
}

func (stream *streamManagerState) deleteUnneededFiles(imageName string,
	subFS, imgFS *filesystem.FileSystem, mountPoint string) error {
	pathsToDelete := make([]string, 0)
	imgHashToInodesTable := imgFS.HashToInodesTable()
	imgFilenameToInodeTable := imgFS.FilenameToInodeTable()
	for pathname, inum := range subFS.FilenameToInodeTable() {
		if inode, ok := subFS.InodeTable[inum].(*filesystem.RegularInode); ok {
			if inode.Size > 0 {
				if _, ok := imgHashToInodesTable[inode.Hash]; !ok {
					pathsToDelete = append(pathsToDelete, pathname)
				}
			} else {
				if _, ok := imgFilenameToInodeTable[pathname]; !ok {
					pathsToDelete = append(pathsToDelete, pathname)
				}
			}
		}
	}
	if len(pathsToDelete) < 1 {
		return nil
	}
	stream.unpacker.logger.Printf("Deleting(%s): %d unneeded files\n",
		imageName, len(pathsToDelete))
	for _, pathname := range pathsToDelete {
		stream.unpacker.logger.Printf("Delete(%s): %s\n", imageName, pathname)
		os.Remove(path.Join(mountPoint, pathname))
	}
	return nil
}

func (stream *streamManagerState) fetch(imageName string,
	objectsToFetch []hash.Hash, destDirname string) error {
	startTime := time.Now()
	stream.streamInfo.status = unpackproto.StatusStreamFetching
	srpcClient, err := stream.unpacker.imageServerResource.GetHTTP(nil, 0)
	if err != nil {
		return err
	}
	defer srpcClient.Put()
	objectServer := objectclient.AttachObjectClient(srpcClient)
	defer objectServer.Close()
	objectsReader, err := objectServer.GetObjects(objectsToFetch)
	if err != nil {
		stream.streamInfo.status = unpackproto.StatusStreamMounted
		return err
	}
	defer objectsReader.Close()
	stream.unpacker.logger.Printf("Fetching(%s) %d objects\n",
		imageName, len(objectsToFetch))
	for _, hashVal := range objectsToFetch {
		length, reader, err := objectsReader.NextObject()
		if err != nil {
			stream.unpacker.logger.Println(err)
			stream.streamInfo.status = unpackproto.StatusStreamMounted
			return err
		}
		err = readOne(destDirname, hashVal, length, reader)
		reader.Close()
		if err != nil {
			stream.unpacker.logger.Println(err)
			stream.streamInfo.status = unpackproto.StatusStreamMounted
			return err
		}
	}
	stream.unpacker.logger.Printf("Fetched(%s) %d objects in %s\n",
		imageName, len(objectsToFetch), format.Duration(time.Since(startTime)))
	return nil
}

func readOne(objectsDir string, hashVal hash.Hash, length uint64,
	reader io.Reader) error {
	filename := path.Join(objectsDir, objectcache.HashToFilename(hashVal))
	dirname := path.Dir(filename)
	if err := os.MkdirAll(dirname, dirPerms); err != nil {
		return err
	}
	return fsutil.CopyToFile(filename, filePerms, reader, length)
}
