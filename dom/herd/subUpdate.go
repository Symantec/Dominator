package herd

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"path"
	"syscall"
	"time"
)

type state struct {
	sub                     *Sub
	subFS                   *filesystem.FileSystem
	requiredFS              *filesystem.FileSystem
	requiredInodeToSubInode map[uint64]uint64
	inodesChanged           map[uint64]bool   // Required inode number.
	inodesCreated           map[uint64]string // Required inode number.
	subObjectCacheUsage     map[hash.Hash]uint64
}

// Returns true if no update needs to be performed.
func (sub *Sub) buildUpdateRequest(request *subproto.UpdateRequest) (
	bool, bool) {
	sub.herd.computeSemaphore <- struct{}{}
	defer func() { <-sub.herd.computeSemaphore }()
	var state state
	state.sub = sub
	state.subFS = sub.fileSystem
	requiredImage := sub.herd.getImageNoError(sub.mdb.RequiredImage)
	state.requiredFS = requiredImage.FileSystem
	filter := requiredImage.Filter
	request.Triggers = requiredImage.Triggers
	state.requiredInodeToSubInode = make(map[uint64]uint64)
	state.inodesChanged = make(map[uint64]bool)
	state.inodesCreated = make(map[uint64]string)
	state.subObjectCacheUsage = make(map[hash.Hash]uint64, len(sub.objectCache))
	var rusageStart, rusageStop syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	// Populate subObjectCacheUsage.
	for _, hash := range sub.objectCache {
		state.subObjectCacheUsage[hash] = 0
	}
	if !filesystem.CompareDirectoriesMetadata(&state.subFS.DirectoryInode,
		&state.requiredFS.DirectoryInode, nil) {
		makeDirectory(request, &state.requiredFS.DirectoryInode, "/", false)
	}
	if compareDirectories(request, &state,
		&state.subFS.DirectoryInode, &state.requiredFS.DirectoryInode,
		"/", filter) {
		return false, true
	}
	// Look for multiply used objects and tell the sub.
	for obj, useCount := range state.subObjectCacheUsage {
		if useCount > 1 {
			if request.MultiplyUsedObjects == nil {
				request.MultiplyUsedObjects = make(map[hash.Hash]uint64)
			}
			request.MultiplyUsedObjects[obj] = useCount
		}
	}
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStop)
	sub.lastComputeUpdateCpuDuration = time.Duration(
		rusageStop.Utime.Sec)*time.Second +
		time.Duration(rusageStop.Utime.Usec)*time.Microsecond -
		time.Duration(rusageStart.Utime.Sec)*time.Second -
		time.Duration(rusageStart.Utime.Usec)*time.Microsecond
	computeCpuTimeDistribution.Add(sub.lastComputeUpdateCpuDuration)
	if len(request.FilesToCopyToCache) > 0 ||
		len(request.InodesToMake) > 0 ||
		len(request.HardlinksToMake) > 0 ||
		len(request.PathsToDelete) > 0 ||
		len(request.DirectoriesToMake) > 0 ||
		len(request.InodesToChange) > 0 {
		sub.herd.logger.Printf(
			"buildUpdateRequest(%s) took: %s user CPU time\n",
			sub, sub.lastComputeUpdateCpuDuration)
		return false, false
	}
	return true, false
}

func compareDirectories(request *subproto.UpdateRequest, state *state,
	subDirectory, requiredDirectory *filesystem.DirectoryInode,
	myPathName string, filter *filter.Filter) bool {
	// First look for entries that should be deleted.
	if filter != nil && subDirectory != nil {
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
		if filter != nil && filter.Match(pathname) {
			continue
		}
		var subEntry *filesystem.DirectoryEntry
		if subDirectory != nil {
			if se, ok := subDirectory.EntriesByName[name]; ok {
				subEntry = se
			}
		}
		requiredInode := requiredEntry.Inode()
		if _, ok := requiredInode.(*filesystem.ComputedRegularInode); ok {
			// Replace with computed file.
			inode, ok := state.sub.computedInodes[pathname]
			if !ok {
				state.sub.herd.logger.Printf(
					"compareDirectories(%s): missing computed file: %s\n",
					state.sub, pathname)
				return true
			}
			newEntry := new(filesystem.DirectoryEntry)
			newEntry.Name = name
			newEntry.InodeNumber = requiredEntry.InodeNumber
			newEntry.SetInode(inode)
			requiredEntry = newEntry
		}
		if subEntry == nil {
			addEntry(request, state, requiredEntry, pathname)
		} else {
			compareEntries(request, state, subEntry, requiredEntry, pathname)
		}
		// If a directory: descend (possibly with the directory for the sub).
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
	return false
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
	subEntry, requiredEntry *filesystem.DirectoryEntry, myPathName string) {
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
		myPathName, state.subFS.InodeToFilenamesTable()[subInum][0])
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
	names := state.requiredFS.InodeToFilenamesTable()[requiredEntry.InodeNumber]
	if len(names) > 1 {
		var sameDataInode filesystem.GenericInode
		var sameDataName string
		for _, name := range names {
			if inum, found := state.subFS.FilenameToInodeTable()[name]; found {
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
			} else {
				// Not in object cache: grab it from file-system.
				if inos, ok := state.subFS.HashToInodesTable()[inode.Hash]; ok {
					var fileToCopy subproto.FileToCopyToCache
					fileToCopy.Name =
						state.subFS.InodeToFilenamesTable()[inos[0]][0]
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
