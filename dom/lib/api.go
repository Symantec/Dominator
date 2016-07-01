package lib

import (
	"errors"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/srpc"
	"log"
)

var (
	ErrorFailedToGetObject = errors.New("get object failed")
)

type Sub struct {
	Hostname       string
	Client         *srpc.Client
	FileSystem     *filesystem.FileSystem
	ComputedInodes map[string]*filesystem.RegularInode
	ObjectCache    objectcache.ObjectCache
}

func BuildMissingLists(sub Sub, image *image.Image, pushComputedFiles bool,
	logger *log.Logger) (
	[]hash.Hash, map[hash.Hash]struct{}) {
	return sub.buildMissingLists(image, pushComputedFiles, logger)
}

func PushObjects(sub Sub, objectsToPush map[hash.Hash]struct{},
	objectServer objectserver.ObjectServer, logger *log.Logger) error {
	return sub.pushObjects(objectsToPush, objectServer, logger)
}

func (sub *Sub) String() string {
	return sub.Hostname
}
