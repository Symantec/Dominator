package image

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
)

type Image struct {
	Filter     *filter.Filter
	FileSystem *filesystem.FileSystem
}
