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
	request.Triggers = requiredImage.Triggers
	var state state
	state.subInodeToRequiredInode = make(map[uint64]uint64)
	compareDirectories(request, &state,
		&subFS.DirectoryInode, &requiredFS.DirectoryInode,
		"/", filter)
	// TODO(rgooch): Implement this.
}

func compareDirectories(request *subproto.UpdateRequest, state *state,
	subDirectory, requiredDirectory *filesystem.DirectoryInode,
	myPathName string, filter *filter.Filter) {
	// First look for entries that should be deleted.
	makeSubDirectory := false
	if subDirectory == nil {
		makeSubDirectory = true
	} else {
		for name, _ := range subDirectory.EntriesByName {
			pathname := path.Join(myPathName, name)
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
			fmt.Printf("Different directory: %s...\n", myPathName) // HACK
			makeSubDirectory = true
			// TODO(rgooch): Update metadata.
		}
	}
	if makeSubDirectory {
		var newdir subproto.Directory
		newdir.Name = myPathName
		newdir.Mode = requiredDirectory.Mode
		newdir.Uid = requiredDirectory.Uid
		newdir.Gid = requiredDirectory.Gid
		request.DirectoriesToMake = append(request.DirectoriesToMake, newdir)
	}
	for name, requiredEntry := range requiredDirectory.EntriesByName {
		pathname := path.Join(myPathName, name)
		if filter.Match(pathname) {
			continue
		}
		if subDirectory == nil {
			compareEntries(request, state, nil, requiredEntry, pathname,
				filter)
		} else {
			if subEntry, ok := subDirectory.EntriesByName[name]; ok {
				compareEntries(request, state, subEntry, requiredEntry,
					pathname, filter)
			} else {
				compareEntries(request, state, nil, requiredEntry,
					pathname, filter)
			}
		}
	}
}

func addEntry(request *subproto.UpdateRequest, state *state,
	requiredEntry *filesystem.DirectoryEntry, myPathName string) {
	requiredInode := requiredEntry.Inode()
	if requiredInode, ok := requiredInode.(*filesystem.DirectoryInode); ok {
		var newdir subproto.Directory
		newdir.Name = myPathName
		newdir.Mode = requiredInode.Mode
		newdir.Uid = requiredInode.Uid
		newdir.Gid = requiredInode.Gid
		request.DirectoriesToMake = append(request.DirectoriesToMake, newdir)
	} else {
		fmt.Printf("Add entry: %s...\n", myPathName) // HACK
	}
}

func compareEntries(request *subproto.UpdateRequest, state *state,
	subEntry, requiredEntry *filesystem.DirectoryEntry,
	myPathName string, filter *filter.Filter) {
	switch requiredInode := requiredEntry.Inode().(type) {
	case *filesystem.RegularInode:
		compareRegularFile(request, state, subEntry,
			requiredInode, requiredEntry.InodeNumber, myPathName)
		return
	case *filesystem.SymlinkInode:
		compareSymlink(request, state, subEntry,
			requiredInode, requiredEntry.InodeNumber, myPathName)
		return
	case *filesystem.Inode:
		compareFile(request, state, subEntry,
			requiredInode, requiredEntry.InodeNumber, myPathName)
		return
	case *filesystem.DirectoryInode:
		compareDirectory(request, state, subEntry, requiredInode, myPathName,
			filter)
		return
	}
	panic("Unsupported entry type")
}

func compareRegularFile(request *subproto.UpdateRequest, state *state,
	subEntry *filesystem.DirectoryEntry,
	requiredInode *filesystem.RegularInode, requiredInodeNumber uint64,
	myPathName string) {
	if subEntry == nil {
		fmt.Printf("Add rfile: %s...\n", myPathName) // HACK
		// TODO(rgooch): Add entry.
		return
	}
	if subInode, ok := subEntry.Inode().(*filesystem.RegularInode); ok {
		if requiredInum, ok :=
			state.subInodeToRequiredInode[subEntry.InodeNumber]; ok {
			if requiredInum != requiredInodeNumber {
				//
				fmt.Printf("Different links: %s...\n", myPathName) // HACK
			}
		} else {
			state.subInodeToRequiredInode[subEntry.InodeNumber] =
				requiredInodeNumber
		}
		sameMetadata := filesystem.CompareRegularInodesMetadata(
			subInode, requiredInode, os.Stdout)
		sameData := filesystem.CompareRegularInodesData(subInode,
			requiredInode, os.Stdout)
		if sameMetadata && sameData {
			return
		}
		fmt.Printf("Different rfile: %s...\n", myPathName) // HACK
	} else {
		fmt.Printf("Delete+add rfile: %s...\n", myPathName) // HACK
	}
	// TODO(rgooch): Delete entry and replace.
}

func compareSymlink(request *subproto.UpdateRequest, state *state,
	subEntry *filesystem.DirectoryEntry,
	requiredInode *filesystem.SymlinkInode, requiredInodeNumber uint64,
	myPathName string) {
	if subInode, ok := subEntry.Inode().(*filesystem.SymlinkInode); ok {
		if requiredInum, ok :=
			state.subInodeToRequiredInode[subEntry.InodeNumber]; ok {
			if requiredInum != requiredInodeNumber {
				fmt.Printf("Different links: %s...\n", myPathName) // HACK
			}
		} else {
			state.subInodeToRequiredInode[subEntry.InodeNumber] =
				requiredInodeNumber
		}
		if filesystem.CompareSymlinkInodes(subInode, requiredInode, os.Stdout) {
			return
		}
		fmt.Printf("Different symlink: %s...\n", myPathName) // HACK
	} else {
		fmt.Printf("Add symlink: %s...\n", myPathName) // HACK
	}
	// TODO(rgooch): Delete entry and replace.
}

func compareFile(request *subproto.UpdateRequest, state *state,
	subEntry *filesystem.DirectoryEntry,
	requiredInode *filesystem.Inode, requiredInodeNumber uint64,
	myPathName string) {
	if subInode, ok := subEntry.Inode().(*filesystem.Inode); ok {
		if requiredInum, ok :=
			state.subInodeToRequiredInode[subEntry.InodeNumber]; ok {
			if requiredInum != requiredInodeNumber {
				fmt.Printf("Different links: %s...\n", myPathName) // HACK
			}
		} else {
			state.subInodeToRequiredInode[subEntry.InodeNumber] =
				requiredInodeNumber
		}
		if filesystem.CompareInodes(subInode, requiredInode, os.Stdout) {
			return
		}
		fmt.Printf("Different file: %s...\n", myPathName) // HACK
	} else {
		fmt.Printf("Add file: %s...\n", myPathName) // HACK
	}
	// TODO(rgooch): Delete entry and replace.
}

func compareDirectory(request *subproto.UpdateRequest, state *state,
	subEntry *filesystem.DirectoryEntry,
	requiredInode *filesystem.DirectoryInode,
	myPathName string, filter *filter.Filter) {
	if subEntry == nil {
		compareDirectories(request, state, nil, requiredInode, myPathName,
			filter)
		return
	}
	if subInode, ok := subEntry.Inode().(*filesystem.DirectoryInode); ok {
		compareDirectories(request, state, subInode, requiredInode,
			myPathName, filter)
	} else {
		compareDirectories(request, state, nil, requiredInode, myPathName,
			filter)
	}
}
