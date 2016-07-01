package lib

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/objectcache"
	"log"
)

type Sub struct {
	Hostname       string
	FileSystem     *filesystem.FileSystem
	ComputedInodes map[string]*filesystem.RegularInode
	ObjectCache    objectcache.ObjectCache
}

func BuildMissingLists(sub Sub, image *image.Image, pushComputedFiles bool,
	logger *log.Logger) (
	[]hash.Hash, map[hash.Hash]struct{}) {
	return sub.buildMissingLists(image, pushComputedFiles, logger)
}

func (sub *Sub) String() string {
	return sub.Hostname
}
