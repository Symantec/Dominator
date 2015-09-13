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
	requiredPathName := path.Join(parentName, requiredDirectory.Name)
	// First look for entries that should be deleted.
	makeSubDirectory := false
	if subDirectory == nil {
		makeSubDirectory = true
	} else {
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
		if !filesystem.CompareDirectoriesMetadata(subDirectory,
			requiredDirectory, os.Stdout) {
			fmt.Printf("Different directory: %s...\n", requiredPathName) // HACK
			makeSubDirectory = true
			// TODO(rgooch): Update metadata.
		}
	}
	if makeSubDirectory {
		var newdir subproto.Directory
		newdir.Name = requiredPathName
		newdir.Mode = uint32(requiredDirectory.Mode)
		newdir.Uid = requiredDirectory.Uid
		newdir.Gid = requiredDirectory.Gid
		request.DirectoriesToMake = append(request.DirectoriesToMake, newdir)
	}
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
		compareSymlink(request, subEntry, re, parentName)
		return
	case *filesystem.File:
		compareFile(request, subEntry, re, parentName)
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
		fmt.Printf("Different rfile: %s...\n", requiredRegularFile.Name) // HACK
	} else {
		fmt.Printf("Add rfile: %s...\n", requiredRegularFile.Name) // HACK
	}
	// TODO(rgooch): Delete entry and replace.
}

func compareSymlink(request *subproto.UpdateRequest,
	subEntry interface{}, requiredSymlink *filesystem.Symlink,
	parentName string) {
	if subSymlink, ok := subEntry.(*filesystem.Symlink); ok {
		if filesystem.CompareSymlinkInodes(subSymlink.Inode(),
			requiredSymlink.Inode(), os.Stdout) {
			return
		}
		fmt.Printf("Different symlink: %s...\n", requiredSymlink.Name) // HACK
	} else {
		fmt.Printf("Add symlink: %s...\n", requiredSymlink.Name) // HACK
	}
	// TODO(rgooch): Delete entry and replace.
}

func compareFile(request *subproto.UpdateRequest,
	subEntry interface{}, requiredFile *filesystem.File,
	parentName string) {
	if subFile, ok := subEntry.(*filesystem.File); ok {
		if filesystem.CompareInodes(subFile.Inode(), requiredFile.Inode(),
			os.Stdout) {
			return
		}
		fmt.Printf("Different file: %s...\n", requiredFile.Name) // HACK
	} else {
		fmt.Printf("Add file: %s...\n", requiredFile.Name) // HACK
	}
	// TODO(rgooch): Delete entry and replace.
}

func compareDirectory(request *subproto.UpdateRequest,
	subEntry interface{}, requiredDirectory *filesystem.Directory,
	parentName string, filter *filter.Filter) {
	if subDirectory, ok := subEntry.(*filesystem.Directory); ok {
		compareDirectories(request, subDirectory, requiredDirectory,
			parentName, filter)
	} else {
		compareDirectories(request, nil, requiredDirectory, parentName, filter)
	}
}
