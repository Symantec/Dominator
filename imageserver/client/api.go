package client

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
)

func AddImage(client *srpc.Client, name string, img *image.Image) error {
	return addImage(client, name, img)
}

func CheckImage(client *srpc.Client, name string) (bool, error) {
	return checkImage(client, name)
}

func ChownDirectory(client *srpc.Client, dirname, ownerGroup string) error {
	return chownDirectory(client, dirname, ownerGroup)
}

func DeleteImage(client *srpc.Client, name string) error {
	return deleteImage(client, name)
}

func GetImage(client *srpc.Client, name string) (*image.Image, error) {
	return getImage(client, name)
}

func ListDirectories(client *srpc.Client) ([]image.Directory, error) {
	return listDirectories(client)
}

func ListImages(client *srpc.Client) ([]string, error) {
	return listImages(client)
}

func ListUnreferencedObjects(client *srpc.Client) (
	map[hash.Hash]uint64, error) {
	return listUnreferencedObjects(client)
}

func MakeDirectory(client *srpc.Client, dirname string) error {
	return makeDirectory(client, dirname)
}
