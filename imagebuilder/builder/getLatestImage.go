package builder

import (
	imageclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/verstr"
	"path"
)

func getLatestImage(client *srpc.Client, imageStream string) (
	string, *image.Image, error) {
	imageNames, err := imageclient.ListImages(client)
	if err != nil {
		return "", nil, err
	}
	verstr.Sort(imageNames)
	imageName := ""
	for _, name := range imageNames {
		dirname := path.Dir(name)
		if dirname == imageStream {
			imageName = name
		}
	}
	if imageName == "" {
		return "", nil, nil
	}
	if img, err := imageclient.GetImage(client, imageName); err != nil {
		return "", nil, err
	} else {
		img.FileSystem.RebuildInodePointers()
		return imageName, img, nil
	}
}
