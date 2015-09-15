package herd

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"os"
	"path"
)

type state struct {
	subInodeToRequiredInode map[uint64]uint64
}

func (sub *Sub) buildUpdateRequest(request *subproto.UpdateRequest) {
	fmt.Println("buildUpdateRequest()") // TODO(rgooch): Delete debugging.
	subFS := sub.fileSystem
	requiredImage := sub.herd.getImage(sub.requiredImage)
	requiredFS := requiredImage.FileSystem
	filter := requiredImage.Filter
	var state state
	state.subInodeToRequiredInode = make(map[uint64]uint64)
	compareDirectories(request, &state, &subFS.Directory, &requiredFS.Directory,
		"", filter)
	// TODO(rgooch): Implement this.
}

func compareDirectories(request *subproto.UpdateRequest, state *state,
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
		newdir.Mode = requiredDirectory.Mode
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
			compareEntries(request, state, nil, requiredEntry, requiredPathName,
				filter)
		} else {
			if subEntry, ok := subDirectory.EntriesByName[name]; ok {
				compareEntries(request, state, subEntry, requiredEntry,
					requiredPathName, filter)
			} else {
				compareEntries(request, state, nil, requiredEntry,
					requiredPathName, filter)
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

func compareEntries(request *subproto.UpdateRequest, state *state,
	subEntry, requiredEntry interface{},
	parentName string, filter *filter.Filter) {
	switch re := requiredEntry.(type) {
	case *filesystem.RegularFile:
		compareRegularFile(request, state, subEntry, re, parentName)
		return
	case *filesystem.Symlink:
		compareSymlink(request, state, subEntry, re, parentName)
		return
	case *filesystem.File:
		compareFile(request, state, subEntry, re, parentName)
		return
	case *filesystem.Directory:
		compareDirectory(request, state, subEntry, re, parentName, filter)
		return
	}
	panic("Unsupported entry type")
}

func compareRegularFile(request *subproto.UpdateRequest, state *state,
	subEntry interface{}, requiredRegularFile *filesystem.RegularFile,
	parentName string) {
	debugFilename := path.Join(parentName, requiredRegularFile.Name)
	if subRegularFile, ok := subEntry.(*filesystem.RegularFile); ok {
		if requiredInode, ok :=
			state.subInodeToRequiredInode[subRegularFile.InodeNumber]; ok {
			if requiredInode == requiredRegularFile.InodeNumber {
				//
				fmt.Printf("Different links: %s...\n", debugFilename) // HACK
			}
		} else {
			state.subInodeToRequiredInode[subRegularFile.InodeNumber] =
				requiredRegularFile.InodeNumber
		}
		sameMetadata := filesystem.CompareRegularInodesMetadata(
			subRegularFile.Inode(), requiredRegularFile.Inode(),
			os.Stdout)
		sameData := filesystem.CompareRegularInodesData(subRegularFile.Inode(),
			requiredRegularFile.Inode(), os.Stdout)
		if sameMetadata && sameData {
			return
		}
		fmt.Printf("Different rfile: %s...\n", debugFilename) // HACK
	} else {
		fmt.Printf("Add rfile: %s...\n", debugFilename) // HACK
	}
	// TODO(rgooch): Delete entry and replace.
}

func compareSymlink(request *subproto.UpdateRequest, state *state,
	subEntry interface{}, requiredSymlink *filesystem.Symlink,
	parentName string) {
	debugFilename := path.Join(parentName, requiredSymlink.Name)
	if subSymlink, ok := subEntry.(*filesystem.Symlink); ok {
		if requiredInode, ok :=
			state.subInodeToRequiredInode[subSymlink.InodeNumber]; ok {
			if requiredInode != requiredSymlink.InodeNumber {
				fmt.Printf("Different links: %s...\n", debugFilename) // HACK
			}
		} else {
			state.subInodeToRequiredInode[subSymlink.InodeNumber] =
				requiredSymlink.InodeNumber
		}
		if filesystem.CompareSymlinkInodes(subSymlink.Inode(),
			requiredSymlink.Inode(), os.Stdout) {
			return
		}
		fmt.Printf("Different symlink: %s...\n", debugFilename) // HACK
	} else {
		fmt.Printf("Add symlink: %s...\n", debugFilename) // HACK
	}
	// TODO(rgooch): Delete entry and replace.
}

func compareFile(request *subproto.UpdateRequest, state *state,
	subEntry interface{}, requiredFile *filesystem.File,
	parentName string) {
	debugFilename := path.Join(parentName, requiredFile.Name)
	if subFile, ok := subEntry.(*filesystem.File); ok {
		if requiredInode, ok :=
			state.subInodeToRequiredInode[subFile.InodeNumber]; ok {
			if requiredInode != requiredFile.InodeNumber {
				fmt.Printf("Different links: %s...\n", debugFilename) // HACK
			}
		} else {
			state.subInodeToRequiredInode[subFile.InodeNumber] =
				requiredFile.InodeNumber
		}
		if filesystem.CompareInodes(subFile.Inode(), requiredFile.Inode(),
			os.Stdout) {
			return
		}
		fmt.Printf("Different file: %s...\n", debugFilename) // HACK
	} else {
		fmt.Printf("Add file: %s...\n", debugFilename) // HACK
	}
	// TODO(rgooch): Delete entry and replace.
}

func compareDirectory(request *subproto.UpdateRequest, state *state,
	subEntry interface{}, requiredDirectory *filesystem.Directory,
	parentName string, filter *filter.Filter) {
	if subDirectory, ok := subEntry.(*filesystem.Directory); ok {
		compareDirectories(request, state, subDirectory, requiredDirectory,
			parentName, filter)
	} else {
		compareDirectories(request, state, nil, requiredDirectory, parentName,
			filter)
	}
}
