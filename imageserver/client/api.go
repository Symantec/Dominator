package client

import (
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/image"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func AddImage(client *srpc.Client, name string, img *image.Image) error {
	return addImage(client, name, img)
}

func AddImageTrusted(client *srpc.Client, name string, img *image.Image) error {
	return addImageTrusted(client, name, img)
}

func ChangeImageExpiration(client *srpc.Client, name string,
	expiresAt time.Time) error {
	return changeImageExpiration(client, name, expiresAt)
}

func CheckDirectory(client *srpc.Client, name string) (bool, error) {
	return checkDirectory(client, name)
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

func DeleteUnreferencedObjects(client *srpc.Client, percentage uint8,
	bytes uint64) error {
	return deleteUnreferencedObjects(client, percentage, bytes)
}

func FindLatestImage(client *srpc.Client, dirname string,
	ignoreExpiring bool) (string, error) {
	return findLatestImage(client, dirname, ignoreExpiring)
}

func GetImage(client *srpc.Client, name string) (*image.Image, error) {
	return getImage(client, name, 0)
}

func GetImageExpiration(client *srpc.Client, name string) (time.Time, error) {
	return getImageExpiration(client, name)
}

func GetImageWithTimeout(client *srpc.Client, name string,
	timeout time.Duration) (*image.Image, error) {
	return getImage(client, name, timeout)
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
