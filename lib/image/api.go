package image

import (
	"time"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/triggers"
)

type Annotation struct {
	Object *hash.Hash // These are mutually exclusive.
	URL    string
}

type DirectoryMetadata struct {
	OwnerGroup string
}

type Directory struct {
	Name     string
	Metadata DirectoryMetadata
}

type Image struct {
	CreatedBy    string // Username. Set by imageserver. Empty: unauthenticated.
	Filter       *filter.Filter
	FileSystem   *filesystem.FileSystem
	Triggers     *triggers.Triggers
	ReleaseNotes *Annotation
	BuildLog     *Annotation
	CreatedOn    time.Time
	ExpiresAt    time.Time
	Packages     []Package
}

type Package struct {
	Name    string
	Size    uint64 // Bytes.
	Version string
}

// ForEachObject will call objectFunc for all objects (including those for
// annotations) for the image. If objectFunc returns a non-nil error, processing
// stops and the error is returned.
func (image *Image) ForEachObject(objectFunc func(hash.Hash) error) error {
	return image.forEachObject(objectFunc)
}

func (image *Image) ListMissingObjects(
	objectsChecker objectserver.ObjectsChecker) ([]hash.Hash, error) {
	return image.listMissingObjects(objectsChecker)
}

// ListObjects will return a list of all objects (including those for
// annotations for an image).
func (image *Image) ListObjects() []hash.Hash {
	return image.listObjects()
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

func SortDirectories(directories []Directory) {
	sortDirectories(directories)
}
