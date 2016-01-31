package image

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/triggers"
)

type Annotation struct {
	Object *hash.Hash // These are mutually exclusive.
	URL    string
}

type Image struct {
	Filter       *filter.Filter
	FileSystem   *filesystem.FileSystem
	Triggers     *triggers.Triggers
	ReleaseNotes *Annotation
	BuildLog     *Annotation
}
