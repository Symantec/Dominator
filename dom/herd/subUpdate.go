package herd

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"os"
	"path"
	"syscall"
	"time"
)

type state struct {
	requiredInodeToSubInode map[uint64]uint64
	inodesChanged           map[uint64]bool // Required inode number.
	subFS                   *filesystem.FileSystem
	requiredFS              *filesystem.FileSystem
}

func (sub *Sub) buildUpdateRequest(request *subproto.UpdateRequest) {
	fmt.Println("buildUpdateRequest()") // TODO(rgooch): Delete debugging.
	var state state
	state.subFS = &sub.fileSystem.FileSystem
	requiredImage := sub.herd.getImage(sub.requiredImage)
	state.requiredFS = requiredImage.FileSystem
	filter := requiredImage.Filter
	request.Triggers = requiredImage.Triggers
	state.requiredInodeToSubInode = make(map[uint64]uint64)
	state.inodesChanged = make(map[uint64]bool)
	var rusageStart, rusageStop syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	compareDirectories(request, &state,
		&state.subFS.DirectoryInode, &state.requiredFS.DirectoryInode,
		"/", filter)
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStop) // HACK
	cpuTime := time.Duration(rusageStop.Utime.Sec)*time.Second +
		time.Duration(rusageStop.Utime.Usec)*time.Microsecond -
		time.Duration(rusageStart.Utime.Sec)*time.Second -
		time.Duration(rusageStart.Utime.Usec)*time.Microsecond
	fmt.Printf("Build update request took: %s user CPU time\n", cpuTime)
}

func compareDirectories(request *subproto.UpdateRequest, state *state,
	subDirectory, requiredDirectory *filesystem.DirectoryInode,
	myPathName string, filter *filter.Filter) {
	// First look for entries that should be deleted.
	if subDirectory != nil {
		for name := range subDirectory.EntriesByName {
			pathname := path.Join(myPathName, name)
			if filter.Match(pathname) {
				continue
			}
			if _, ok := requiredDirectory.EntriesByName[name]; !ok {
				request.PathsToDelete = append(request.PathsToDelete, pathname)
				fmt.Printf("Delete: %s\n", pathname) // HACK
			}
		}
	}
	for name, requiredEntry := range requiredDirectory.EntriesByName {
		pathname := path.Join(myPathName, name)
		if filter.Match(pathname) {
			continue
		}
		var subEntry *filesystem.DirectoryEntry
		if subDirectory != nil {
			if se, ok := subDirectory.EntriesByName[name]; ok {
				subEntry = se
			}
		}
		if subEntry == nil {
			addEntry(request, state, requiredEntry, pathname)
		} else {
			compareEntries(request, state, subEntry, requiredEntry, pathname,
				filter)
		}
		// If a directory: descend (possibly with the directory for the sub).
		requiredInode := requiredEntry.Inode()
		if requiredInode, ok := requiredInode.(*filesystem.DirectoryInode); ok {
			var subInode *filesystem.DirectoryInode
			if subEntry != nil {
				if si, ok := subEntry.Inode().(*filesystem.DirectoryInode); ok {
					subInode = si
				}
			}
			compareDirectories(request, state, subInode, requiredInode,
				pathname, filter)
		}
	}
}

func addEntry(request *subproto.UpdateRequest, state *state,
	requiredEntry *filesystem.DirectoryEntry, myPathName string) {
	requiredInode := requiredEntry.Inode()
	if requiredInode, ok := requiredInode.(*filesystem.DirectoryInode); ok {
		makeDirectory(request, requiredInode, myPathName, true)
	} else {
		addInode(request, state, requiredEntry, myPathName)
	}
}

func compareEntries(request *subproto.UpdateRequest, state *state,
	subEntry, requiredEntry *filesystem.DirectoryEntry,
	myPathName string, filter *filter.Filter) {
	var sameType, sameMetadata, sameData bool
	switch requiredInode := requiredEntry.Inode().(type) {
	case *filesystem.RegularInode:
		sameType, sameMetadata, sameData =
			compareRegularFile(request, state, subEntry, requiredInode,
				myPathName)
	case *filesystem.SymlinkInode:
		sameType, sameMetadata, sameData =
			compareSymlink(request, state, subEntry, requiredInode, myPathName)
	case *filesystem.Inode:
		sameType, sameMetadata, sameData =
			compareFile(request, state, subEntry, requiredInode, myPathName)
	case *filesystem.DirectoryInode:
		compareDirectory(request, state, subEntry, requiredInode, myPathName,
			filter)
		return
	default:
		panic("Unsupported entry type")
	}
	if sameType && sameData && sameMetadata {
		relink(request, state, subEntry, requiredEntry, myPathName)
		return
	}
	if sameType && sameData {
		updateMetadata(request, state, subEntry, requiredEntry, myPathName)
		relink(request, state, subEntry, requiredEntry, myPathName)
		return
	}
	request.PathsToDelete = append(request.PathsToDelete, myPathName)
	addInode(request, state, requiredEntry, myPathName)
}

func compareRegularFile(request *subproto.UpdateRequest, state *state,
	subEntry *filesystem.DirectoryEntry, requiredInode *filesystem.RegularInode,
	myPathName string) (sameType, sameMetadata, sameData bool) {
	if subInode, ok := subEntry.Inode().(*filesystem.RegularInode); ok {
		sameType = true
		sameMetadata = filesystem.CompareRegularInodesMetadata(
			subInode, requiredInode, nil)
		sameData = filesystem.CompareRegularInodesData(subInode,
			requiredInode, os.Stdout)
	}
	return
}

func compareSymlink(request *subproto.UpdateRequest, state *state,
	subEntry *filesystem.DirectoryEntry, requiredInode *filesystem.SymlinkInode,
	myPathName string) (sameType, sameMetadata, sameData bool) {
	if subInode, ok := subEntry.Inode().(*filesystem.SymlinkInode); ok {
		sameType = true
		sameMetadata = filesystem.CompareSymlinkInodesMetadata(subInode,
			requiredInode, nil)
		sameData = filesystem.CompareSymlinkInodesData(subInode, requiredInode,
			os.Stdout)
	}
	return
}

func compareFile(request *subproto.UpdateRequest, state *state,
	subEntry *filesystem.DirectoryEntry, requiredInode *filesystem.Inode,
	myPathName string) (sameType, sameMetadata, sameData bool) {
	if subInode, ok := subEntry.Inode().(*filesystem.Inode); ok {
		sameType = true
		sameMetadata = filesystem.CompareInodesMetadata(subInode, requiredInode,
			nil)
		sameData = filesystem.CompareInodesData(subInode, requiredInode,
			os.Stdout)
	}
	return
}

func compareDirectory(request *subproto.UpdateRequest, state *state,
	subEntry *filesystem.DirectoryEntry,
	requiredInode *filesystem.DirectoryInode,
	myPathName string, filter *filter.Filter) {
	if subInode, ok := subEntry.Inode().(*filesystem.DirectoryInode); ok {
		if filesystem.CompareDirectoriesMetadata(subInode, requiredInode, nil) {
			return
		}
		makeDirectory(request, requiredInode, myPathName, false)
	} else {
		request.PathsToDelete = append(request.PathsToDelete, myPathName)
		makeDirectory(request, requiredInode, myPathName, true)
		fmt.Printf("Replace non-directory: %s...\n", myPathName) // HACK
	}
}

func relink(request *subproto.UpdateRequest, state *state,
	subEntry, requiredEntry *filesystem.DirectoryEntry, myPathName string) {
	subInum, ok := state.requiredInodeToSubInode[requiredEntry.InodeNumber]
	if !ok {
		state.requiredInodeToSubInode[requiredEntry.InodeNumber] =
			subEntry.InodeNumber
		return
	}
	if subInum == subEntry.InodeNumber {
		return
	}
	var hardlink subproto.Hardlink
	hardlink.Source = myPathName
	hardlink.Target = state.subFS.InodeToFilenamesTable[subInum][0]
	request.HardlinksToMake = append(request.HardlinksToMake, hardlink)
	fmt.Printf("Make link: %s => %s\n", hardlink.Source,
		hardlink.Target) // HACK
}

func updateMetadata(request *subproto.UpdateRequest, state *state,
	subEntry, requiredEntry *filesystem.DirectoryEntry, myPathName string) {
	if changed := state.inodesChanged[requiredEntry.InodeNumber]; changed {
		return
	}
	var inode subproto.Inode
	inode.Name = myPathName
	inode.GenericInode = requiredEntry.Inode()
	request.InodesToChange = append(request.InodesToChange, inode)
	state.inodesChanged[requiredEntry.InodeNumber] = true
	fmt.Printf("Update metadata: %s\n", myPathName) // HACK
}

func makeDirectory(request *subproto.UpdateRequest,
	requiredInode *filesystem.DirectoryInode, pathName string, create bool) {
	var newdir subproto.Directory
	newdir.Name = pathName
	newdir.Mode = requiredInode.Mode
	newdir.Uid = requiredInode.Uid
	newdir.Gid = requiredInode.Gid
	if create {
		request.DirectoriesToMake = append(request.DirectoriesToMake, newdir)
		fmt.Printf("Add directory: %s...\n", pathName) // HACK
	} else {
		request.DirectoriesToChange = append(request.DirectoriesToMake, newdir)
		fmt.Printf("Change directory: %s...\n", pathName) // HACK
	}
}

func addInode(request *subproto.UpdateRequest, state *state,
	requiredEntry *filesystem.DirectoryEntry, myPathName string) {
	fmt.Printf("Add entry: %s...\n", myPathName) // HACK
	// TODO(rgooch): Add entry.
}
