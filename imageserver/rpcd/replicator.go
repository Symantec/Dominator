package rpcd

import (
	"errors"
	"fmt"
	"io"
	"time"

	imageclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) replicator(finishedReplication chan<- struct{}) {
	initialTimeout := time.Second * 15
	timeout := initialTimeout
	var nextSleepStopTime time.Time
	for {
		nextSleepStopTime = time.Now().Add(timeout)
		if client, err := srpc.DialHTTP("tcp", t.replicationMaster,
			timeout); err != nil {
			t.logger.Printf("Error dialing: %s %s\n", t.replicationMaster, err)
		} else {
			if conn, err := client.Call(
				"ImageServer.GetImageUpdates"); err != nil {
				t.logger.Println(err)
			} else {
				if err := t.getUpdates(conn, &finishedReplication); err != nil {
					if err == io.EOF {
						t.logger.Println(
							"Connection to image replicator closed")
						if nextSleepStopTime.Sub(time.Now()) < 1 {
							timeout = initialTimeout
						}
					} else {
						t.logger.Println(err)
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

func (t *srpcType) getUpdates(conn *srpc.Conn,
	finishedReplication *chan<- struct{}) error {
	t.logger.Printf("Image replicator: connected to: %s\n", t.replicationMaster)
	replicationStartTime := time.Now()
	initialImages := make(map[string]struct{})
	if t.archiveMode {
		initialImages = nil
	}
	for {
		var imageUpdate imageserver.ImageUpdate
		if err := conn.Decode(&imageUpdate); err != nil {
			if err == io.EOF {
				return err
			}
			return errors.New("decode err: " + err.Error())
		}
		switch imageUpdate.Operation {
		case imageserver.OperationAddImage:
			if imageUpdate.Name == "" {
				if initialImages != nil {
					t.deleteMissingImages(initialImages)
					initialImages = nil
				}
				if *finishedReplication != nil {
					close(*finishedReplication)
					*finishedReplication = nil
				}
				t.logger.Printf("Replicated all current images in %s\n",
					format.Duration(time.Since(replicationStartTime)))
				continue
			}
			if initialImages != nil {
				initialImages[imageUpdate.Name] = struct{}{}
			}
			if err := t.addImage(imageUpdate.Name); err != nil {
				return errors.New("error adding image: " + imageUpdate.Name +
					": " + err.Error())
			}
		case imageserver.OperationDeleteImage:
			if t.archiveMode {
				continue
			}
			t.logger.Printf("Replicator(%s): delete image\n", imageUpdate.Name)
			if err := t.imageDataBase.DeleteImage(imageUpdate.Name,
				nil); err != nil {
				return err
			}
		case imageserver.OperationMakeDirectory:
			directory := imageUpdate.Directory
			if directory == nil {
				return errors.New("nil imageUpdate.Directory")
			}
			if err := t.imageDataBase.UpdateDirectory(*directory); err != nil {
				return err
			}
		}
	}
}

func (t *srpcType) deleteMissingImages(imagesToKeep map[string]struct{}) {
	missingImages := make([]string, 0)
	for _, imageName := range t.imageDataBase.ListImages() {
		if _, ok := imagesToKeep[imageName]; !ok {
			missingImages = append(missingImages, imageName)
		}
	}
	for _, imageName := range missingImages {
		t.logger.Printf("Replicator(%s): delete missing image\n", imageName)
		if err := t.imageDataBase.DeleteImage(imageName, nil); err != nil {
			t.logger.Println(err)
		}
	}
}

func (t *srpcType) extendImageExpiration(name string,
	img *image.Image) (bool, error) {
	timeout := time.Second * 60
	client, err := t.imageserverResource.GetHTTP(nil, timeout)
	if err != nil {
		return false, err
	}
	defer client.Put()
	expiresAt, err := imageclient.GetImageExpiration(client, name)
	if err != nil {
		if err == io.EOF {
			client.Close()
		}
		return false, err
	}
	return t.imageDataBase.ChangeImageExpiration(name, expiresAt,
		&srpc.AuthInformation{HaveMethodAccess: true})
}

func (t *srpcType) addImage(name string) error {
	timeout := time.Second * 60
	if t.checkImageBeingInjected(name) {
		return nil
	}
	logger := prefixlogger.New(fmt.Sprintf("Replicator(%s): ", name), t.logger)
	if img := t.imageDataBase.GetImage(name); img != nil {
		if img.ExpiresAt.IsZero() {
			return nil
		}
		if changed, err := t.extendImageExpiration(name, img); err != nil {
			logger.Println(err)
		} else if changed {
			logger.Println("extended expiration time")
		}
		return nil
	}
	logger.Println("add image")
	client, err := t.imageserverResource.GetHTTP(nil, timeout)
	if err != nil {
		return err
	}
	defer client.Put()
	request := imageserver.GetImageRequest{
		ImageName: name,
		Timeout:   timeout,
	}
	if t.archiveMode && !*archiveExpiringImages {
		request.IgnoreFilesystemIfExpiring = true
	}
	var reply imageserver.GetImageResponse
	err = client.RequestReply("ImageServer.GetImage", request, &reply)
	if err != nil {
		client.Close()
		return err
	}
	img := reply.Image
	if img == nil {
		return errors.New(name + ": not found")
	}
	logger.Println("downloaded image")
	if t.archiveMode && !img.ExpiresAt.IsZero() && !*archiveExpiringImages {
		logger.Println("ignoring expiring image in archiver mode")
		return nil
	}
	img.FileSystem.RebuildInodePointers()
	err = t.imageDataBase.DoWithPendingImage(img, func() error {
		if err := t.getMissingObjects(img, client, logger); err != nil {
			client.Close()
			return err
		}
		err := t.imageDataBase.AddImage(img, name,
			&srpc.AuthInformation{HaveMethodAccess: true})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	logger.Println("added image")
	return nil
}

func (t *srpcType) checkImageBeingInjected(name string) bool {
	t.imagesBeingInjectedLock.Lock()
	defer t.imagesBeingInjectedLock.Unlock()
	_, ok := t.imagesBeingInjected[name]
	return ok
}

func (t *srpcType) getMissingObjects(img *image.Image, client *srpc.Client,
	logger log.DebugLogger) error {
	objClient := objectclient.AttachObjectClient(client)
	defer objClient.Close()
	return img.GetMissingObjects(t.objSrv, objClient, logger)
}
