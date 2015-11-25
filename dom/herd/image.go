package herd

import (
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/proto/imageserver"
	"net/rpc"
	"time"
)

func (herd *Herd) getImage(name string) *image.Image {
	if name == "" {
		return nil
	}
	herd.RLock()
	image := herd.imagesByName[name]
	herd.RUnlock()
	if image != nil {
		return image
	}
	herd.Lock()
	defer herd.Unlock()
	return herd.getImageHaveLock(name)
}

func (herd *Herd) getImageHaveLock(name string) *image.Image {
	if name == "" {
		return nil
	}
	if image := herd.imagesByName[name]; image != nil {
		return image
	}
	// Image not yet known. If it was recently found to be missing, report it
	// as missing. This avoids hammering the imageserver with "are we there
	// yet?", "are we there yet?", "are we there yet?" queries.
	if lastCheck, ok := herd.missingImages[name]; ok {
		if time.Since(lastCheck).Seconds() < 1 {
			return nil
		}
	}
	connection, err := rpc.DialHTTP("tcp", herd.imageServerAddress)
	if err != nil {
		herd.logger.Println(err)
		return nil
	}
	defer connection.Close()
	var request imageserver.GetImageRequest
	request.ImageName = name
	var reply imageserver.GetImageResponse
	err = connection.Call("ImageServer.GetImage", request, &reply)
	if err != nil {
		herd.logger.Printf("Error calling\t%s\n", err)
		return nil
	}
	if reply.Image == nil {
		herd.missingImages[name] = time.Now()
	} else {
		if err := reply.Image.FileSystem.RebuildInodePointers(); err != nil {
			herd.logger.Printf("Error building inode pointers for image: %s %s",
				name, err)
			return nil
		}
		delete(herd.missingImages, name)
		reply.Image.FileSystem.BuildEntryMap()
		reply.Image.FileSystem.BuildInodeToFilenamesTable()
		reply.Image.FileSystem.BuildHashToInodesTable()
		herd.imagesByName[name] = reply.Image
		herd.logger.Printf("Got image: %s\n", name)
	}
	return reply.Image
}
