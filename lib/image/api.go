package image

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/triggers"
)

type Image struct {
	Filter     *filter.Filter
	FileSystem *filesystem.FileSystem
	Triggers   *triggers.Triggers
}
