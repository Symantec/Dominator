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
	if subDirectory != nil {
		subPathName := path.Join(parentName, subDirectory.Name)
		for name, subEntry := range subDirectory.EntriesByName {
			pathname := path.Join(subPathName, entryName(subEntry))
			if filter.Match(pathname) {
				continue
			}
			if _, ok := requiredDirectory.EntriesByName[name]; !ok {
				request.PathsToDelete = append(request.PathsToDelete, pathname)
				fmt.Printf("Delete: %s\n", pathname) // HACK
			}
		}
	}
	requiredPathName := path.Join(parentName, requiredDirectory.Name)
	for name, requiredEntry := range requiredDirectory.EntriesByName {
		pathname := path.Join(requiredPathName, entryName(requiredEntry))
		if filter.Match(pathname) {
			continue
		}
		if subDirectory == nil {
			compareEntries(request, nil, requiredEntry, requiredPathName,
				filter)
		} else {
			if subEntry, ok := subDirectory.EntriesByName[name]; ok {
				compareEntries(request, subEntry, requiredEntry,
					requiredPathName, filter)
			} else {
				compareEntries(request, nil, requiredEntry, requiredPathName,
					filter)
			}
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
	switch re := requiredEntry.(type) {
	case *filesystem.RegularFile:
		compareRegularFile(request, subEntry, re, parentName)
		return
	case *filesystem.Symlink:
		//compareSymlink(request, subEntry, re, parentName)
		return
	case *filesystem.File:
		//compareFile(request, subEntry, re, parentName)
		return
	case *filesystem.Directory:
		compareDirectory(request, subEntry, re, parentName, filter)
		return
	}
	panic("Unsupported entry type")
}

func compareRegularFile(request *subproto.UpdateRequest,
	subEntry interface{}, requiredRegularFile *filesystem.RegularFile,
	parentName string) {
	if subRegularFile, ok := subEntry.(*filesystem.RegularFile); ok {
		sameMetadata := filesystem.CompareRegularInodesMetadata(
			subRegularFile.Inode(), requiredRegularFile.Inode(),
			os.Stdout)
		sameData := filesystem.CompareRegularInodesData(subRegularFile.Inode(),
			requiredRegularFile.Inode(), os.Stdout)
		if sameMetadata && sameData {
			return
		}
		fmt.Printf("Different: %s...\n", subRegularFile.Name) // HACK
	} else {
		fmt.Printf("Add: %s...\n", subRegularFile.Name) // HACK
	}
	// TODO(rgooch): Delete regular file and replace.
}

func compareDirectory(request *subproto.UpdateRequest,
	subEntry interface{}, requiredDirectory *filesystem.Directory,
	parentName string, filter *filter.Filter) {
	if subDirectory, ok := subEntry.(*filesystem.Directory); ok {
		compareDirectories(request, subDirectory, requiredDirectory,
			parentName, filter)
	} else {
		// TODO(rgooch): Delete directory and replace.
	}
}
