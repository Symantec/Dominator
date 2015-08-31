package herd

import (
	"fmt"
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
	image := herd.imagesByName[name]
	if image != nil {
		return image
	}
	connection, err := rpc.DialHTTP("tcp", herd.imageServerAddress)
	if err != nil {
		fmt.Printf("Error dialing\t%s\n", err)
		return nil
	}
	defer connection.Close()
	var request imageserver.GetImageRequest
	request.ImageName = name
	var reply imageserver.GetImageResponse
	err = connection.Call("ImageServer.GetImage", request, &reply)
	if err != nil {
		fmt.Printf("Error calling\t%s\n", err)
		return nil
	}
	// TODO(rgooch): Delete debugging output.
	if reply.Image != nil {
		fmt.Printf("Got image: %s\n", name)
	}
	herd.imagesByName[name] = reply.Image
	return reply.Image
}
