package image

import (
	"github.com/Symantec/Dominator/lib/filesystem"
)

type FilterEntry string

type Filter []FilterEntry

type Image struct {
	Filter     Filter
	FileSystem *filesystem.FileSystem
}
