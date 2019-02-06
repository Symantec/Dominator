package builder

import (
	"fmt"
	"io"
	"time"

	imageclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
)

func getLatestImage(client *srpc.Client, imageStream string,
	buildLog io.Writer) (string, *image.Image, error) {
	imageName, err := imageclient.FindLatestImage(client, imageStream, false)
	if err != nil {
		return "", nil, err
	}
	if imageName == "" {
		return "", nil, nil
	}
	if img, err := getImage(client, imageName, buildLog); err != nil {
		return "", nil, err
	} else {
		return imageName, img, nil
	}
}

func getImage(client *srpc.Client, imageName string, buildLog io.Writer) (
	*image.Image, error) {
	startTime := time.Now()
	if img, err := imageclient.GetImage(client, imageName); err != nil {
		return nil, err
	} else {
		startRebuildTime := time.Now()
		img.FileSystem.RebuildInodePointers()
		finishedTime := time.Now()
		fmt.Fprintf(buildLog, "Downloaded %s in %s, rebuilt pointers in %s\n",
			imageName,
			format.Duration(startRebuildTime.Sub(startTime)),
			format.Duration(finishedTime.Sub(startRebuildTime)))
		return img, nil
	}
}
