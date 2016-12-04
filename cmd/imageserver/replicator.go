package main

import (
	"encoding/gob"
	"errors"
	imgclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/imageserver/scanner"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/hash"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	fsdriver "github.com/Symantec/Dominator/lib/objectserver/filesystem"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"io"
	"log"
	"strings"
	"time"
)

func replicator(address string, imdb *scanner.ImageDataBase,
	objSrv *fsdriver.ObjectServer, archiveMode bool, logger *log.Logger) {
	initialTimeout := time.Second * 15
	timeout := initialTimeout
	var nextSleepStopTime time.Time
	for {
		nextSleepStopTime = time.Now().Add(timeout)
		if client, err := srpc.DialHTTP("tcp", address, timeout); err != nil {
			logger.Printf("Error dialling: %s %s\n", address, err)
		} else {
			if conn, err := client.Call(
				"ImageServer.GetImageUpdates"); err != nil {
				logger.Println(err)
			} else {
				if err := getUpdates(address, conn, imdb, objSrv, archiveMode,
					logger); err != nil {
					if err == io.EOF {
						logger.Println("Connection to image replicator closed")
						if nextSleepStopTime.Sub(time.Now()) < 1 {
							timeout = initialTimeout
						}
					} else {
						logger.Println(err)
					}
				}
				conn.Close()
			}
			client.Close()
		}
		time.Sleep(nextSleepStopTime.Sub(time.Now()))
		if timeout < time.Minute {
			timeout *= 2
		}
	}
}

func getUpdates(address string, conn *srpc.Conn, imdb *scanner.ImageDataBase,
	objSrv *fsdriver.ObjectServer, archiveMode bool, logger *log.Logger) error {
	logger.Printf("Image replicator: connected to: %s\n", address)
	replicationStartTime := time.Now()
	decoder := gob.NewDecoder(conn)
	initialImages := make(map[string]struct{})
	if archiveMode {
		initialImages = nil
	}
	for {
		var imageUpdate imageserver.ImageUpdate
		if err := decoder.Decode(&imageUpdate); err != nil {
			if err == io.EOF {
				return err
			}
			return errors.New("decode err: " + err.Error())
		}
		switch imageUpdate.Operation {
		case imageserver.OperationAddImage:
			if imageUpdate.Name == "" {
				if initialImages != nil {
					deleteMissingImages(imdb, initialImages, logger)
					initialImages = nil
				}
				logger.Printf("Replicated all current images in %s\n",
					format.Duration(time.Since(replicationStartTime)))
				continue
			}
			if initialImages != nil {
				initialImages[imageUpdate.Name] = struct{}{}
			}
			if err := addImage(address, imdb, objSrv, imageUpdate.Name,
				logger); err != nil {
				return err
			}
		case imageserver.OperationDeleteImage:
			if archiveMode {
				continue
			}
			logger.Printf("Replicator(%s): delete image\n", imageUpdate.Name)
			if err := imdb.DeleteImage(imageUpdate.Name, nil); err != nil {
				return err
			}
		case imageserver.OperationMakeDirectory:
			directory := imageUpdate.Directory
			if directory == nil {
				return errors.New("nil imageUpdate.Directory")
			}
			if err := imdb.UpdateDirectory(*directory); err != nil {
				return err
			}
		}
	}
}

func deleteMissingImages(imdb *scanner.ImageDataBase,
	imagesToKeep map[string]struct{}, logger *log.Logger) {
	missingImages := make([]string, 0)
	for _, imageName := range imdb.ListImages() {
		if _, ok := imagesToKeep[imageName]; !ok {
			missingImages = append(missingImages, imageName)
		}
	}
	for _, imageName := range missingImages {
		logger.Printf("Replicator(%s): delete missing image\n", imageName)
		if err := imdb.DeleteImage(imageName, nil); err != nil {
			logger.Println(err)
		}
	}
}

func addImage(address string, imdb *scanner.ImageDataBase,
	objSrv *fsdriver.ObjectServer, name string, logger *log.Logger) error {
	timeout := time.Second * 60
	if imdb.CheckImage(name) {
		return nil
	}
	logger.Printf("Replicator(%s): add image\n", name)
	client, err := srpc.DialHTTP("tcp", address, timeout)
	if err != nil {
		return err
	}
	defer client.Close()
	img, err := imgclient.GetImage(client, name)
	if err != nil {
		return err
	}
	if img == nil {
		return errors.New(name + ": not found")
	}
	logger.Printf("Replicator(%s): downloaded image\n", name)
	if *archiveMode && !img.ExpiresAt.IsZero() && !*archiveExpiringImages {
		logger.Printf(
			"Replicator(%s): ignoring expiring image in archiver mode\n",
			name)
		return nil
	}
	img.FileSystem.RebuildInodePointers()
	if err := getMissingObjectsRetry(address, imdb, objSrv, img.FileSystem,
		logger); err != nil {
		return err
	}
	if err := imdb.AddImage(img, name, nil); err != nil {
		return err
	}
	logger.Printf("Replicator(%s): added image\n", name)
	return nil
}

func getMissingObjectsRetry(address string, imdb *scanner.ImageDataBase,
	objSrv *fsdriver.ObjectServer, fs *filesystem.FileSystem,
	logger *log.Logger) error {
	err := getMissingObjects(address, objSrv, fs, logger)
	if err == nil {
		return nil
	}
	if !strings.Contains(err.Error(), "no space left on device") {
		return err
	}
	logger.Println(err)
	if !deleteUnreferencedObjects(imdb, objSrv, fs, false, logger) {
		return err
	}
	logger.Println(
		"Replicator: retrying after deleting 10% of unreferenced objects")
	err = getMissingObjects(address, objSrv, fs, logger)
	if err == nil {
		return nil
	}
	if !strings.Contains(err.Error(), "no space left on device") {
		return err
	}
	if !deleteUnreferencedObjects(imdb, objSrv, fs, true, logger) {
		return err
	}
	logger.Println(
		"Replicator: retrying after deleting remaining unreferenced objects")
	return getMissingObjects(address, objSrv, fs, logger)
}

func getMissingObjects(address string, objSrv *fsdriver.ObjectServer,
	fs *filesystem.FileSystem, logger *log.Logger) error {
	hashes := make([]hash.Hash, 0, fs.NumRegularInodes)
	for _, inode := range fs.InodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			if inode.Size > 0 {
				hashes = append(hashes, inode.Hash)
			}
		}
	}
	objectSizes, err := objSrv.CheckObjects(hashes)
	if err != nil {
		return err
	}
	missingObjects := make([]hash.Hash, 0)
	for index, size := range objectSizes {
		if size < 1 {
			missingObjects = append(missingObjects, hashes[index])
		}
	}
	if len(missingObjects) < 1 {
		return nil
	}
	logger.Printf("Replicator: downloading %d of %d objects\n",
		len(missingObjects), len(hashes))
	objClient := objectclient.NewObjectClient(address)
	defer objClient.Close()
	objectsReader, err := objClient.GetObjects(missingObjects)
	if err != nil {
		return err
	}
	defer objectsReader.Close()
	for _, hash := range missingObjects {
		length, reader, err := objectsReader.NextObject()
		if err != nil {
			return err
		}
		_, _, err = objSrv.AddObject(reader, length, &hash)
		reader.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteUnreferencedObjects(imdb *scanner.ImageDataBase,
	objSrv *fsdriver.ObjectServer, fs *filesystem.FileSystem, all bool,
	logger *log.Logger) bool {
	objectsMap := imdb.ListUnreferencedObjects()
	for _, inode := range fs.InodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			delete(objectsMap, inode.Hash)
		}
	}
	numToDelete := len(objectsMap)
	if !all {
		numToDelete = numToDelete / 10
		if numToDelete < 1 {
			numToDelete = numToDelete
		}
	}
	if numToDelete < 1 {
		return false
	}
	count := 0
	var unreferencedBytes uint64
	for hashVal, size := range objectsMap {
		if err := objSrv.DeleteObject(hashVal); err != nil {
			logger.Printf("Error deleting unreferenced object: %x\n", hashVal)
			return false
		}
		unreferencedBytes += size
		count++
		if count >= numToDelete {
			break
		}
	}
	logger.Printf("Deleted %d unreferenced objects consuming %s\n",
		numToDelete, format.FormatBytes(unreferencedBytes))
	return true
}
