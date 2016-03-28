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

// Verify will perform some self-consistency checks on the image. If a problem
// is found, an error is returned.
func (image *Image) Verify() error {
	return image.verify()
}

// VerifyRequiredPaths will verify if required paths are present in the image.
// The table of required paths is given by requiredPaths. If the image is a
// sparse image (has no filter), then this check is skipped. If a problem is
// found, an error is returned.
// This function will create cache data associated with the image, consuming
// more memory.
func (image *Image) VerifyRequiredPaths(requiredPaths map[string]rune) error {
	return image.verifyRequiredPaths(requiredPaths)
}
