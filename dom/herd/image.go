package herd

import (
	"github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"time"
)

func (herd *Herd) getImageNoError(name string) *image.Image {
	image, _ := herd.getImage(name)
	return image
}

func (herd *Herd) getImage(name string) (*image.Image, error) {
	if name == "" {
		return nil, nil
	}
	herd.RLock()
	image := herd.imagesByName[name]
	herd.RUnlock()
	if image != nil {
		return image, nil
	}
	herd.Lock()
	defer herd.Unlock()
	return herd.getImageHaveLock(name)
}

func (herd *Herd) getImageHaveLock(name string) (*image.Image, error) {
	if name == "" {
		return nil, nil
	}
	if image := herd.imagesByName[name]; image != nil {
		return image, nil
	}
	// Image not yet known. If it was recently found to be missing, report it
	// as missing. This avoids hammering the imageserver with "are we there
	// yet?", "are we there yet?", "are we there yet?" queries.
	if lastCheck, ok := herd.missingImages[name]; ok {
		if time.Since(lastCheck.lastGetAttempt).Seconds() < 1 {
			return nil, lastCheck.err
		}
	}
	imageClient, err := srpc.DialHTTP("tcp", herd.imageServerAddress, 0)
	if err != nil {
		herd.missingImages[name] = missingImage{time.Now(), err}
		herd.logger.Println(err)
		return nil, err
	}
	defer imageClient.Close()
	var request imageserver.GetImageRequest
	request.ImageName = name
	var reply imageserver.GetImageResponse
	err = client.CallGetImage(imageClient, request, &reply)
	if err != nil {
		herd.missingImages[name] = missingImage{time.Now(), err}
		herd.logger.Printf("Error calling\t%s\n", err)
		return nil, err
	}
	if reply.Image == nil {
		herd.missingImages[name] = missingImage{time.Now(), nil}
	} else {
		if err := reply.Image.FileSystem.RebuildInodePointers(); err != nil {
			herd.logger.Printf("Error building inode pointers for image: %s %s",
				name, err)
			return nil, err
		}
		delete(herd.missingImages, name)
		reply.Image.FileSystem.BuildEntryMap()
		herd.imagesByName[name] = reply.Image
		herd.logger.Printf("Got image: %s\n", name)
	}
	return reply.Image, nil
}
