package herd

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"path"
	"syscall"
	"time"
)

type state struct {
	subFS                   *filesystem.FileSystem
	requiredFS              *filesystem.FileSystem
	requiredInodeToSubInode map[uint64]uint64
	inodesChanged           map[uint64]bool   // Required inode number.
	inodesCreated           map[uint64]string // Required inode number.
	subFilenameToInode      map[string]uint64
	subObjectCacheUsage     map[hash.Hash]uint64
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
	state.inodesCreated = make(map[uint64]string)
	state.subObjectCacheUsage = make(map[hash.Hash]uint64,
		len(sub.fileSystem.ObjectCache))
	var rusageStart, rusageStop syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	// Populate subObjectCacheUsage.
	for _, hash := range sub.fileSystem.ObjectCache {
		state.subObjectCacheUsage[hash] = 0
	}
	compareDirectories(request, &state,
		&state.subFS.DirectoryInode, &state.requiredFS.DirectoryInode,
		"/", filter)
	// Look for multiply used objects and tell the sub.
	for obj, useCount := range state.subObjectCacheUsage {
		if useCount > 1 {
			if request.MultiplyUsedObjects == nil {
				request.MultiplyUsedObjects = make(map[hash.Hash]uint64)
			}
			request.MultiplyUsedObjects[obj] = useCount
			fmt.Printf("%d uses of object: %x\n", useCount, obj) // HACK
		}
	}
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStop)
	sub.lastComputeUpdateCpuDuration = time.Duration(
		rusageStop.Utime.Sec)*time.Second +
		time.Duration(rusageStop.Utime.Usec)*time.Microsecond -
		time.Duration(rusageStart.Utime.Sec)*time.Second -
		time.Duration(rusageStart.Utime.Usec)*time.Microsecond
	sub.herd.logger.Printf(
		"buildUpdateRequest(%s) took: %s user CPU time\n",
		sub.hostname, sub.lastComputeUpdateCpuDuration)
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
	subInode := subEntry.Inode()
	requiredInode := requiredEntry.Inode()
	sameType, sameMetadata, sameData := filesystem.CompareInodes(
		subInode, requiredInode, nil)
	if requiredInode, ok := requiredInode.(*filesystem.DirectoryInode); ok {
		if sameMetadata {
			return
		}
		if sameType {
			makeDirectory(request, requiredInode, myPathName, false)
		} else {
			request.PathsToDelete = append(request.PathsToDelete, myPathName)
			makeDirectory(request, requiredInode, myPathName, true)
		}
		return
	}
	if sameType && sameData && sameMetadata {
		relink(request, state, subEntry, requiredEntry, myPathName)
		return
	}
	if sameType && sameData {
		updateMetadata(request, state, requiredEntry, myPathName)
		relink(request, state, subEntry, requiredEntry, myPathName)
		return
	}
	request.PathsToDelete = append(request.PathsToDelete, myPathName)
	addInode(request, state, requiredEntry, myPathName)
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
	makeHardlink(request,
		myPathName, state.subFS.InodeToFilenamesTable[subInum][0])
}

func makeHardlink(request *subproto.UpdateRequest, newLink, target string) {
	var hardlink subproto.Hardlink
	hardlink.NewLink = newLink
	hardlink.Target = target
	request.HardlinksToMake = append(request.HardlinksToMake, hardlink)
}

func updateMetadata(request *subproto.UpdateRequest, state *state,
	requiredEntry *filesystem.DirectoryEntry, myPathName string) {
	if state.inodesChanged[requiredEntry.InodeNumber] {
		return
	}
	var inode subproto.Inode
	inode.Name = myPathName
	inode.GenericInode = requiredEntry.Inode()
	request.InodesToChange = append(request.InodesToChange, inode)
	state.inodesChanged[requiredEntry.InodeNumber] = true
}

func makeDirectory(request *subproto.UpdateRequest,
	requiredInode *filesystem.DirectoryInode, pathName string, create bool) {
	var newInode subproto.Inode
	newInode.Name = pathName
	var newDirectoryInode filesystem.DirectoryInode
	newDirectoryInode.Mode = requiredInode.Mode
	newDirectoryInode.Uid = requiredInode.Uid
	newDirectoryInode.Gid = requiredInode.Gid
	newInode.GenericInode = &newDirectoryInode
	if create {
		request.DirectoriesToMake = append(request.DirectoriesToMake, newInode)
	} else {
		request.InodesToChange = append(request.InodesToChange, newInode)
	}
}

func addInode(request *subproto.UpdateRequest, state *state,
	requiredEntry *filesystem.DirectoryEntry, myPathName string) {
	requiredInode := requiredEntry.Inode()
	if name, ok := state.inodesCreated[requiredEntry.InodeNumber]; ok {
		makeHardlink(request, myPathName, name)
		return
	}
	// Try to find a sibling inode.
	names := state.requiredFS.InodeToFilenamesTable[requiredEntry.InodeNumber]
	if len(names) > 1 {
		var sameDataInode filesystem.GenericInode
		var sameDataName string
		for _, name := range names {
			if inum, found := state.getSubInodeFromFilename(name); found {
				subInode := state.subFS.InodeTable[inum]
				_, sameMetadata, sameData := filesystem.CompareInodes(
					subInode, requiredInode, nil)
				if sameMetadata && sameData {
					makeHardlink(request, myPathName, name)
					return
				}
				if sameData {
					sameDataInode = subInode
					sameDataName = name
				}
			}
		}
		if sameDataInode != nil {
			updateMetadata(request, state, requiredEntry, sameDataName)
			makeHardlink(request, myPathName, sameDataName)
			return
		}
	}
	if inode, ok := requiredEntry.Inode().(*filesystem.RegularInode); ok {
		if inode.Size > 0 {
			if _, ok := state.subObjectCacheUsage[inode.Hash]; ok {
				state.subObjectCacheUsage[inode.Hash]++
				if state.subObjectCacheUsage[inode.Hash] > 1 {
					fmt.Printf("Duplicate use of hash for: %s\n",
						myPathName) // HACK
				}
			} else {
				// Not in object cache: grab it from file-system.
				if state.subFS.HashToInodesTable == nil {
					state.subFS.BuildHashToInodesTable()
				}
				if ilist, ok := state.subFS.HashToInodesTable[inode.Hash]; ok {
					var fileToCopy subproto.FileToCopyToCache
					fileToCopy.Name =
						state.subFS.InodeToFilenamesTable[ilist[0]][0]
					fileToCopy.Hash = inode.Hash
					request.FilesToCopyToCache = append(
						request.FilesToCopyToCache, fileToCopy)
					state.subObjectCacheUsage[inode.Hash] = 1
				} else {
					panic("No object in cache for: " + myPathName)
				}
			}
		}
	}
	var inode subproto.Inode
	inode.Name = myPathName
	inode.GenericInode = requiredEntry.Inode()
	request.InodesToMake = append(request.InodesToMake, inode)
	state.inodesCreated[requiredEntry.InodeNumber] = myPathName
}

func (state *state) getSubInodeFromFilename(name string) (uint64, bool) {
	if state.subFilenameToInode == nil {
		fmt.Println("Making subFilenameToInode map...") // HACK
		state.subFilenameToInode = make(map[string]uint64)
		for inum, names := range state.subFS.InodeToFilenamesTable {
			for _, n := range names {
				state.subFilenameToInode[n] = inum
			}
		}
		fmt.Println("Made subFilenameToInode map") // HACK
	}
	inum, ok := state.subFilenameToInode[name]
	return inum, ok
}
