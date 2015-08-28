package herd

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/proto/imageserver"
	"net/rpc"
)

func (herd *Herd) getImage(name string) *image.Image {
	herd.Lock()
	defer herd.Unlock()
	if herd.imagesByName == nil {
		herd.imagesByName = make(map[string]*image.Image)
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
	var request imageserver.GetImageRequest
	request.ImageName = name
	var reply imageserver.GetImageResponse
	err = connection.Call("ImageServer.GetImage", request, &reply)
	if err != nil {
		fmt.Printf("Error calling\t%s\n", err)
		return nil
	}
	connection.Close()
	// TODO(rgooch): Delete debugging output.
	if reply.Image != nil {
		fmt.Printf("Got image: %s\n", name)
	}
	herd.imagesByName[name] = reply.Image
	return reply.Image
}
