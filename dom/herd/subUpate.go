package herd

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	subproto "github.com/Symantec/Dominator/proto/sub"
)

func (sub *Sub) buildUpdateRequest(request *subproto.UpdateRequest) {
	subFS := sub.fileSystem
	requiredImage := sub.herd.getImage(sub.requiredImage)
	requiredFS := requiredImage.FileSystem
	filter := requiredImage.Filter
	compareDirectories(&subFS.Directory, &requiredFS.Directory, filter)
	// TODO(rgooch): Implement this.
}

func compareDirectories(subDirectory, requiredDirectory *filesystem.Directory,
	filter *filter.Filter) {
	// First look for entries that should be deleted.

}
