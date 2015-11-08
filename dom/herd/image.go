package herd

import (
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/proto/imageserver"
	"net/rpc"
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
	if reply.Image != nil {
		if err := reply.Image.FileSystem.RebuildInodePointers(); err != nil {
			herd.logger.Printf("Error building inode pointers for image: %s %s",
				name, err)
			return nil
		}
		reply.Image.FileSystem.BuildEntryMap()
		reply.Image.FileSystem.BuildInodeToFilenamesTable()
		reply.Image.FileSystem.BuildHashToInodesTable()
		herd.imagesByName[name] = reply.Image
		herd.logger.Printf("Got image: %s\n", name)
	}
	return reply.Image
}
