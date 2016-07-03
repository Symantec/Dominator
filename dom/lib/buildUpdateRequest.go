package lib

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"log"
	"path"
	"time"
)

// Returns true if there is a failure due to missing computed files.
func (sub *Sub) buildUpdateRequest(image *image.Image,
	request *subproto.UpdateRequest, deleteMissingComputedFiles bool,
	logger *log.Logger) bool {
	sub.requiredFS = image.FileSystem
	filter := image.Filter
	request.Triggers = image.Triggers
	sub.requiredInodeToSubInode = make(map[uint64]uint64)
	sub.inodesChanged = make(map[uint64]bool)
	sub.inodesCreated = make(map[uint64]string)
	sub.subObjectCacheUsage = make(map[hash.Hash]uint64, len(sub.ObjectCache))
	// Populate subObjectCacheUsage.
	for _, hash := range sub.ObjectCache {
		sub.subObjectCacheUsage[hash] = 0
	}
	if !filesystem.CompareDirectoriesMetadata(&sub.FileSystem.DirectoryInode,
		&sub.requiredFS.DirectoryInode, nil) {
		makeDirectory(request, &sub.requiredFS.DirectoryInode, "/", false)
	}
	if sub.compareDirectories(request,
		&sub.FileSystem.DirectoryInode, &sub.requiredFS.DirectoryInode,
		"/", filter, deleteMissingComputedFiles, logger) {
		return true
	}
	// Look for multiply used objects and tell the sub.
	for obj, useCount := range sub.subObjectCacheUsage {
		if useCount > 1 {
			if request.MultiplyUsedObjects == nil {
				request.MultiplyUsedObjects = make(map[hash.Hash]uint64)
			}
			request.MultiplyUsedObjects[obj] = useCount
		}
	}
	return false
}

// Returns true if there is a failure due to missing computed files.
func (sub *Sub) compareDirectories(request *subproto.UpdateRequest,
	subDirectory, requiredDirectory *filesystem.DirectoryInode,
	myPathName string, filter *filter.Filter, deleteMissingComputedFiles bool,
	logger *log.Logger) bool {
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
			inode, ok := sub.ComputedInodes[pathname]
			if !ok {
				if deleteMissingComputedFiles {
					if subEntry != nil {
						request.PathsToDelete = append(request.PathsToDelete,
							pathname)
					}
					continue
				}
				logger.Printf(
					"compareDirectories(%s): missing computed file: %s\n",
					sub, pathname)
				return true
			}
			setComputedFileMtime(inode, subEntry)
			newEntry := new(filesystem.DirectoryEntry)
			newEntry.Name = name
			newEntry.InodeNumber = requiredEntry.InodeNumber
			newEntry.SetInode(inode)
			requiredEntry = newEntry
		}
		if subEntry == nil {
			sub.addEntry(request, requiredEntry, pathname)
		} else {
			sub.compareEntries(request, subEntry, requiredEntry, pathname)
		}
		// If a directory: descend (possibly with the directory for the sub).
		if requiredInode, ok := requiredInode.(*filesystem.DirectoryInode); ok {
			var subInode *filesystem.DirectoryInode
			if subEntry != nil {
				if si, ok := subEntry.Inode().(*filesystem.DirectoryInode); ok {
					subInode = si
				}
			}
			sub.compareDirectories(request, subInode, requiredInode, pathname,
				filter, deleteMissingComputedFiles, logger)
		}
	}
	return false
}

func setComputedFileMtime(requiredInode *filesystem.RegularInode,
	subEntry *filesystem.DirectoryEntry) {
	if requiredInode.MtimeSeconds >= 0 {
		return
	}
	if subEntry != nil {
		subInode := subEntry.Inode()
		if subInode, ok := subInode.(*filesystem.RegularInode); ok {
			if requiredInode.Hash == subInode.Hash {
				requiredInode.MtimeNanoSeconds = subInode.MtimeNanoSeconds
				requiredInode.MtimeSeconds = subInode.MtimeSeconds
				return
			}
		}
	}
	requiredInode.MtimeSeconds = time.Now().Unix()
}

func (sub *Sub) addEntry(request *subproto.UpdateRequest,
	requiredEntry *filesystem.DirectoryEntry, myPathName string) {
	requiredInode := requiredEntry.Inode()
	if requiredInode, ok := requiredInode.(*filesystem.DirectoryInode); ok {
		makeDirectory(request, requiredInode, myPathName, true)
	} else {
		sub.addInode(request, requiredEntry, myPathName)
	}
}

func (sub *Sub) compareEntries(request *subproto.UpdateRequest,
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
		sub.relink(request, subEntry, requiredEntry, myPathName)
		return
	}
	if sameType && sameData {
		sub.updateMetadata(request, requiredEntry, myPathName)
		sub.relink(request, subEntry, requiredEntry, myPathName)
		return
	}
	sub.addInode(request, requiredEntry, myPathName)
}

func (sub *Sub) relink(request *subproto.UpdateRequest,
	subEntry, requiredEntry *filesystem.DirectoryEntry, myPathName string) {
	subInum, ok := sub.requiredInodeToSubInode[requiredEntry.InodeNumber]
	if !ok {
		sub.requiredInodeToSubInode[requiredEntry.InodeNumber] =
			subEntry.InodeNumber
		return
	}
	if subInum == subEntry.InodeNumber {
		return
	}
	makeHardlink(request,
		myPathName, sub.FileSystem.InodeToFilenamesTable()[subInum][0])
}

func makeHardlink(request *subproto.UpdateRequest, newLink, target string) {
	var hardlink subproto.Hardlink
	hardlink.NewLink = newLink
	hardlink.Target = target
	request.HardlinksToMake = append(request.HardlinksToMake, hardlink)
}

func (sub *Sub) updateMetadata(request *subproto.UpdateRequest,
	requiredEntry *filesystem.DirectoryEntry, myPathName string) {
	if sub.inodesChanged[requiredEntry.InodeNumber] {
		return
	}
	var inode subproto.Inode
	inode.Name = myPathName
	inode.GenericInode = requiredEntry.Inode()
	request.InodesToChange = append(request.InodesToChange, inode)
	sub.inodesChanged[requiredEntry.InodeNumber] = true
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

func (sub *Sub) addInode(request *subproto.UpdateRequest,
	requiredEntry *filesystem.DirectoryEntry, myPathName string) {
	requiredInode := requiredEntry.Inode()
	if name, ok := sub.inodesCreated[requiredEntry.InodeNumber]; ok {
		makeHardlink(request, myPathName, name)
		return
	}
	// Try to find a sibling inode.
	names := sub.requiredFS.InodeToFilenamesTable()[requiredEntry.InodeNumber]
	subFS := sub.FileSystem
	if len(names) > 1 {
		var sameDataInode filesystem.GenericInode
		var sameDataName string
		for _, name := range names {
			if inum, found := subFS.FilenameToInodeTable()[name]; found {
				subInode := sub.FileSystem.InodeTable[inum]
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
			sub.updateMetadata(request, requiredEntry, sameDataName)
			makeHardlink(request, myPathName, sameDataName)
			return
		}
	}
	if inode, ok := requiredEntry.Inode().(*filesystem.RegularInode); ok {
		if inode.Size > 0 {
			if _, ok := sub.subObjectCacheUsage[inode.Hash]; ok {
				sub.subObjectCacheUsage[inode.Hash]++
			} else {
				// Not in object cache: grab it from file-system.
				if inos, ok := subFS.HashToInodesTable()[inode.Hash]; ok {
					var fileToCopy subproto.FileToCopyToCache
					fileToCopy.Name =
						sub.FileSystem.InodeToFilenamesTable()[inos[0]][0]
					fileToCopy.Hash = inode.Hash
					request.FilesToCopyToCache = append(
						request.FilesToCopyToCache, fileToCopy)
					sub.subObjectCacheUsage[inode.Hash] = 1
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
	sub.inodesCreated[requiredEntry.InodeNumber] = myPathName
}
