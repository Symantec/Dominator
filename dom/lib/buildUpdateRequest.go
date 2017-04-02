package lib

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/debuglogger"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"path"
	"sort"
	"time"
)

// Returns true if there is a failure due to missing computed files.
func (sub *Sub) buildUpdateRequest(image *image.Image,
	request *subproto.UpdateRequest, deleteMissingComputedFiles bool,
	ignoreMissingComputedFiles bool, slogger log.Logger) bool {
	logger := debuglogger.Upgrade(slogger)
	sub.requiredFS = image.FileSystem
	sub.filter = image.Filter
	request.Triggers = image.Triggers
	sub.requiredInodeToSubInode = make(map[uint64]uint64)
	sub.inodesMapped = make(map[uint64]struct{})
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
		"/", deleteMissingComputedFiles, ignoreMissingComputedFiles, logger) {
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
	myPathName string, deleteMissingComputedFiles bool,
	ignoreMissingComputedFiles bool, logger log.DebugLogger) bool {
	// First look for entries that should be deleted.
	if sub.filter != nil && subDirectory != nil {
		for name := range subDirectory.EntriesByName {
			pathname := path.Join(myPathName, name)
			if sub.filter.Match(pathname) {
				continue
			}
			if _, ok := requiredDirectory.EntriesByName[name]; !ok {
				request.PathsToDelete = append(request.PathsToDelete, pathname)
			}
		}
	}
	// For the love of repeatable unit tests, sort before looping.
	names := make([]string, 0, len(requiredDirectory.EntriesByName))
	for name := range requiredDirectory.EntriesByName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		requiredEntry := requiredDirectory.EntriesByName[name]
		pathname := path.Join(myPathName, name)
		if sub.filter != nil && sub.filter.Match(pathname) {
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
				if ignoreMissingComputedFiles {
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
			sub.addEntry(request, requiredEntry, pathname, logger)
		} else {
			sub.compareEntries(request, subEntry, requiredEntry, pathname,
				logger)
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
				deleteMissingComputedFiles, ignoreMissingComputedFiles, logger)
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
	requiredEntry *filesystem.DirectoryEntry, myPathName string,
	logger log.DebugLogger) {
	requiredInode := requiredEntry.Inode()
	if requiredInode, ok := requiredInode.(*filesystem.DirectoryInode); ok {
		makeDirectory(request, requiredInode, myPathName, true)
	} else {
		sub.addInode(request, requiredEntry, myPathName, logger)
	}
}

func (sub *Sub) compareEntries(request *subproto.UpdateRequest,
	subEntry, requiredEntry *filesystem.DirectoryEntry, myPathName string,
	logger log.DebugLogger) {
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
		if sub.relink(request, subEntry, requiredEntry, myPathName, logger) {
			logger.Debugf(0, "identical relink OK for %s\n", myPathName)
			return
		}
	} else if sameType && sameData {
		if sub.relink(request, subEntry, requiredEntry, myPathName,
			logger) {
			logger.Debugf(0, "relink OK for %s\n", myPathName)
			sub.updateMetadata(request, requiredEntry, myPathName)
			return
		}
	}
	sub.addInode(request, requiredEntry, myPathName, logger)
}

func (sub *Sub) relink(request *subproto.UpdateRequest,
	subEntry, requiredEntry *filesystem.DirectoryEntry,
	myPathName string, logger log.DebugLogger) bool {
	subInum, ok := sub.requiredInodeToSubInode[requiredEntry.InodeNumber]
	if !ok {
		if _, mapped := sub.inodesMapped[subEntry.InodeNumber]; mapped {
			return false
		}
		sub.requiredInodeToSubInode[requiredEntry.InodeNumber] =
			subEntry.InodeNumber
		logger.Debugf(0, "mapping sub inum: %d for: %s\n",
			subEntry.InodeNumber, myPathName)
		sub.inodesMapped[subEntry.InodeNumber] = struct{}{}
		return true
	}
	if subInum == subEntry.InodeNumber {
		return true
	}
	makeHardlink(request,
		myPathName, sub.FileSystem.InodeToFilenamesTable()[subInum][0])
	return true
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
	requiredEntry *filesystem.DirectoryEntry, myPathName string,
	logger log.DebugLogger) {
	requiredInode := requiredEntry.Inode()
	logger.Debugf(0, "addInode(%s, %d) Uid=%d\n",
		myPathName, requiredEntry.InodeNumber, requiredInode.GetUid())
	if name, ok := sub.inodesCreated[requiredEntry.InodeNumber]; ok {
		logger.Debugf(0, "make link: %s to %s\n", myPathName, name)
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
			if name == myPathName {
				logger.Debugf(0, "skipping self comparison: %s\n", name)
				continue
			}
			if inum, found := subFS.FilenameToInodeTable()[name]; found {
				subInode := sub.FileSystem.InodeTable[inum]
				_, sameMetadata, sameData := filesystem.CompareInodes(
					subInode, requiredInode, nil)
				if sameMetadata && sameData {
					logger.Debugf(0, "make sibling link: %s to %s (uid=%d)\n",
						myPathName, name, subInode.GetUid())
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
			logger.Debugf(0, "same data make link: %s to %s (uid=%d)\n",
				myPathName, sameDataName, sameDataInode.GetUid())
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
				logger.Debugf(0, "copy to cache for: %s\n", myPathName)
				request.FilesToCopyToCache = append(
					request.FilesToCopyToCache,
					sub.getFileToCopy(myPathName, inode.Hash))
				sub.subObjectCacheUsage[inode.Hash] = 1
			}
		}
	}
	var inode subproto.Inode
	inode.Name = myPathName
	inode.GenericInode = requiredEntry.Inode()
	request.InodesToMake = append(request.InodesToMake, inode)
	sub.inodesCreated[requiredEntry.InodeNumber] = myPathName
}

func (sub *Sub) getFileToCopy(myPathName string,
	hashVal hash.Hash) subproto.FileToCopyToCache {
	subFS := sub.FileSystem
	requiredFS := sub.requiredFS
	inos, ok := subFS.HashToInodesTable()[hashVal]
	if !ok {
		panic("No object in cache for: " + myPathName)
	}
	file := subproto.FileToCopyToCache{
		Name: subFS.InodeToFilenamesTable()[inos[0]][0],
		Hash: hashVal,
	}
	// Try to find an inode where all its links will be deleted and mark one of
	// the links (filenames) to be hardlinked instead of copied into the cache.
	for _, iNum := range inos {
		filenames := subFS.InodeToFilenamesTable()[iNum]
		for _, filename := range filenames {
			if _, ok := requiredFS.FilenameToInodeTable()[filename]; ok {
				filenames = nil
				break
			}
			if sub.filter == nil || sub.filter.Match(filename) {
				filenames = nil
				break
			}
		}
		if filenames != nil {
			file.DoHardlink = true
			break
		}
	}
	return file
}
