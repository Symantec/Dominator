package image

import (
	"github.com/Symantec/Dominator/lib/filesystem"
)

type Filter struct {
	FilterLines []string
}

type Image struct {
	Filter     *Filter
	FileSystem *filesystem.FileSystem
}
