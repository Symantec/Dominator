package main

import (
	"encoding/gob"
	"errors"
	imgclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/imageserver/scanner"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/lib/srpc"
	fsdriver "github.com/Symantec/Dominator/objectserver/filesystem"
	"github.com/Symantec/Dominator/proto/imageserver"
	"io"
	"log"
	"time"
)

func replicator(address string, imdb *scanner.ImageDataBase,
	objSrv *fsdriver.ObjectServer, logger *log.Logger) {
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
				if err := getUpdates(address, conn, imdb, objSrv,
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
	objSrv *fsdriver.ObjectServer, logger *log.Logger) error {
	logger.Printf("Image replicator: connected to: %s\n", address)
	decoder := gob.NewDecoder(conn)
	initialImages := make(map[string]struct{})
	for {
		var imageUpdate imageserver.ImageUpdate
		if err := decoder.Decode(&imageUpdate); err != nil {
			if err == io.EOF {
				return err
			}
			return errors.New("decode err: " + err.Error())
		}
		if imageUpdate.Name == "" {
			if initialImages != nil {
				deleteMissingImages(imdb, initialImages, logger)
				initialImages = nil
			}
			continue
		}
		switch imageUpdate.Operation {
		case imageserver.OperationAddImage:
			if initialImages != nil {
				initialImages[imageUpdate.Name] = struct{}{}
			}
			if err := addImage(address, imdb, objSrv, imageUpdate.Name,
				logger); err != nil {
				return err
			}
		case imageserver.OperationDeleteImage:
			logger.Printf("Replicator(%s): delete image\n", imageUpdate.Name)
			if err := imdb.DeleteImage(imageUpdate.Name); err != nil {
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
		if err := imdb.DeleteImage(imageName); err != nil {
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
	var request imageserver.GetImageRequest
	request.ImageName = name
	var reply imageserver.GetImageResponse
	if err := imgclient.CallGetImage(client, request, &reply); err != nil {
		return err
	}
	if reply.Image == nil {
		return errors.New(name + ": not found")
	}
	logger.Printf("Replicator(%s): downloaded image\n", name)
	reply.Image.FileSystem.RebuildInodePointers()
	if err := getMissingObjects(address, objSrv, reply.Image.FileSystem,
		logger); err != nil {
		return err
	}
	if err := imdb.AddImage(reply.Image, name); err != nil {
		return err
	}
	logger.Printf("Replicator(%s): added image\n", name)
	return nil
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
