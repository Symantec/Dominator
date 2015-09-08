package herd

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"os"
	"path"
)

func (sub *Sub) buildUpdateRequest(request *subproto.UpdateRequest) {
	fmt.Println("buildUpdateRequest()") // TODO(rgooch): Delete debugging.
	subFS := sub.fileSystem
	requiredImage := sub.herd.getImage(sub.requiredImage)
	requiredFS := requiredImage.FileSystem
	filter := requiredImage.Filter
	compareDirectories(request, &subFS.Directory, &requiredFS.Directory, "",
		filter)
	// TODO(rgooch): Implement this.
}

func compareDirectories(request *subproto.UpdateRequest,
	subDirectory, requiredDirectory *filesystem.Directory,
	parentName string, filter *filter.Filter) {
	// First look for entries that should be deleted.
	subPathName := path.Join(parentName, subDirectory.Name)
	for name, subEntry := range subDirectory.EntriesByName {
		pathname := path.Join(subPathName, entryName(subEntry))
		if filter.Match(pathname) {
			continue
		}
		if requiredEntry, ok := requiredDirectory.EntriesByName[name]; ok {
			compareEntries(request, subEntry, requiredEntry, subPathName,
				filter)
		} else {
			request.PathsToDelete = append(request.PathsToDelete, pathname)
		}
	}
}

func entryName(entry interface{}) string {
	switch e := entry.(type) {
	case *filesystem.RegularFile:
		return e.Name
	case *filesystem.Symlink:
		return e.Name
	case *filesystem.File:
		return e.Name
	case *filesystem.Directory:
		return e.Name
	}
	panic("Unsupported entry type")
}

func compareEntries(request *subproto.UpdateRequest,
	subEntry, requiredEntry interface{},
	parentName string, filter *filter.Filter) {
	switch se := subEntry.(type) {
	case *filesystem.RegularFile:
		compareSubRegularFileWithEntry(request, se, requiredEntry)
		return
	case *filesystem.Symlink:
		return
	case *filesystem.File:
		return
	case *filesystem.Directory:
		compareSubDirectoryWithEntry(request, se, requiredEntry, parentName,
			filter)
		return
	}
	panic("Unsupported entry type")
}

func compareSubRegularFileWithEntry(request *subproto.UpdateRequest,
	subRegularFile *filesystem.RegularFile, requiredEntry interface{}) {
	if requiredRegularFile, ok := requiredEntry.(*filesystem.RegularFile); ok {
		if filesystem.CompareRegularFiles(subRegularFile, requiredRegularFile,
			os.Stdout) {
			return
		}
		fmt.Printf("Different: %s...\n", subRegularFile.Name) // HACK
	}
	// TODO(rgooch): Delete regular file and replace.
}

func compareSubDirectoryWithEntry(request *subproto.UpdateRequest,
	subDirectory *filesystem.Directory, requiredEntry interface{},
	parentName string, filter *filter.Filter) {
	if requiredDirectory, ok := requiredEntry.(*filesystem.Directory); ok {
		compareDirectories(request, subDirectory, requiredDirectory,
			parentName,
			filter)
	} else {
		// TODO(rgooch): Delete directory and replace.
	}
}
