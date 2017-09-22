package lib

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/triggers"
	"github.com/Symantec/Dominator/proto/sub"
	"io"
	"os"
	"path"
	"strings"
	"syscall"
	"time"
)

func (t *uType) update(request sub.UpdateRequest,
	oldTriggers *triggers.Triggers) error {
	if request.Triggers == nil {
		request.Triggers = triggers.New()
	}
	t.copyFilesToCache(request.FilesToCopyToCache)
	t.makeObjectCopies(request.MultiplyUsedObjects)
	if t.runTriggers != nil &&
		oldTriggers != nil && len(oldTriggers.Triggers) > 0 {
		t.makeDirectories(request.DirectoriesToMake,
			oldTriggers, false)
		t.makeInodes(request.InodesToMake, request.MultiplyUsedObjects,
			oldTriggers, false)
		t.makeHardlinks(request.HardlinksToMake, oldTriggers, false)
		t.doDeletes(request.PathsToDelete, oldTriggers, false)
		t.changeInodes(request.InodesToChange, oldTriggers, false)
		matchedOldTriggers := oldTriggers.GetMatchedTriggers()
		if t.runTriggers(matchedOldTriggers, "stop", t.logger) {
			t.hadTriggerFailures = true
		}
	}
	fsChangeStartTime := time.Now()
	t.makeDirectories(request.DirectoriesToMake, request.Triggers, true)
	t.makeInodes(request.InodesToMake, request.MultiplyUsedObjects,
		request.Triggers, true)
	t.makeHardlinks(request.HardlinksToMake, request.Triggers, true)
	t.doDeletes(request.PathsToDelete, request.Triggers, true)
	t.changeInodes(request.InodesToChange, request.Triggers, true)
	t.fsChangeDuration = time.Since(fsChangeStartTime)
	matchedNewTriggers := request.Triggers.GetMatchedTriggers()
	if t.runTriggers != nil &&
		t.runTriggers(matchedNewTriggers, "start", t.logger) {
		t.hadTriggerFailures = true
	}
	return t.lastError
}

func (t *uType) copyFilesToCache(filesToCopyToCache []sub.FileToCopyToCache) {
	for _, fileToCopy := range filesToCopyToCache {
		sourcePathname := path.Join(t.rootDirectoryName, fileToCopy.Name)
		destPathname := path.Join(t.objectsDir,
			objectcache.HashToFilename(fileToCopy.Hash))
		prefix := "Copied"
		if fileToCopy.DoHardlink {
			prefix = "Hardlinked"
		}
		if err := copyFile(destPathname, sourcePathname,
			fileToCopy.DoHardlink); err != nil {
			t.lastError = err
			t.logger.Println(err)
		} else {
			t.logger.Printf("%s: %s to cache\n", prefix, sourcePathname)
		}
	}
}

func copyFile(destPathname, sourcePathname string, doHardlink bool) error {
	dirname := path.Dir(destPathname)
	if err := os.MkdirAll(dirname, syscall.S_IRWXU); err != nil {
		return err
	}
	if doHardlink {
		return fsutil.ForceLink(sourcePathname, destPathname)
	}
	sourceFile, err := os.Open(sourcePathname)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	destFile, err := os.Create(destPathname)
	if err != nil {
		return err
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (t *uType) makeObjectCopies(multiplyUsedObjects map[hash.Hash]uint64) {
	for hash, numCopies := range multiplyUsedObjects {
		if numCopies < 2 {
			continue
		}
		objectPathname := path.Join(t.objectsDir,
			objectcache.HashToFilename(hash))
		for numCopies--; numCopies > 0; numCopies-- {
			ext := fmt.Sprintf("~%d~", numCopies)
			if err := copyFile(objectPathname+ext, objectPathname,
				false); err != nil {
				t.lastError = err
				t.logger.Println(err)
			} else {
				t.logger.Printf("Copied object: %x%s\n", hash, ext)
			}
		}
	}
}

func (t *uType) makeInodes(inodesToMake []sub.Inode,
	multiplyUsedObjects map[hash.Hash]uint64, triggers *triggers.Triggers,
	takeAction bool) {
	for _, inode := range inodesToMake {
		fullPathname := path.Join(t.rootDirectoryName, inode.Name)
		triggers.Match(inode.Name)
		if takeAction {
			var err error
			switch inode := inode.GenericInode.(type) {
			case *filesystem.RegularInode:
				err = makeRegularInode(fullPathname, inode, multiplyUsedObjects,
					t.objectsDir, t.logger)
			case *filesystem.SymlinkInode:
				err = makeSymlinkInode(fullPathname, inode, t.logger)
			case *filesystem.SpecialInode:
				err = makeSpecialInode(fullPathname, inode, t.logger)
			}
			if err != nil {
				t.lastError = err
			}
		}
	}
}

func makeRegularInode(fullPathname string,
	inode *filesystem.RegularInode, multiplyUsedObjects map[hash.Hash]uint64,
	objectsDir string, logger log.Logger) error {
	var objectPathname string
	if inode.Size > 0 {
		objectPathname = path.Join(objectsDir,
			objectcache.HashToFilename(inode.Hash))
		numCopies := multiplyUsedObjects[inode.Hash]
		if numCopies > 1 {
			numCopies--
			objectPathname += fmt.Sprintf("~%d~", numCopies)
			if numCopies < 2 {
				delete(multiplyUsedObjects, inode.Hash)
			} else {
				multiplyUsedObjects[inode.Hash] = numCopies
			}
		}
	} else {
		objectPathname = fmt.Sprintf("%s.empty.%d", fullPathname, os.Getpid())
		if file, err := os.OpenFile(objectPathname,
			os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600); err != nil {
			return err
		} else {
			file.Close()
		}
	}
	if err := fsutil.ForceRename(objectPathname, fullPathname); err != nil {
		logger.Println(err)
		return err
	}
	if err := inode.WriteMetadata(fullPathname); err != nil {
		logger.Println(err)
		return err
	} else {
		if inode.Size > 0 {
			logger.Printf("Made inode: %s from: %x\n",
				fullPathname, inode.Hash)
		} else {
			logger.Printf("Made empty inode: %s\n", fullPathname)
		}
	}
	return nil
}

func makeSymlinkInode(fullPathname string,
	inode *filesystem.SymlinkInode, logger log.Logger) error {
	if err := inode.Write(fullPathname); err != nil {
		logger.Println(err)
		return err
	}
	logger.Printf("Made symlink inode: %s -> %s\n", fullPathname, inode.Symlink)
	return nil
}

func makeSpecialInode(fullPathname string, inode *filesystem.SpecialInode,
	logger log.Logger) error {
	if err := inode.Write(fullPathname); err != nil {
		logger.Println(err)
		return err
	}
	logger.Printf("Made special inode: %s\n", fullPathname)
	return nil
}

func (t *uType) makeHardlinks(hardlinksToMake []sub.Hardlink,
	triggers *triggers.Triggers, takeAction bool) {
	tmpName := path.Join(t.objectsDir, "temporaryHardlink")
	for _, hardlink := range hardlinksToMake {
		triggers.Match(hardlink.NewLink)
		if takeAction {
			targetPathname := path.Join(t.rootDirectoryName, hardlink.Target)
			linkPathname := path.Join(t.rootDirectoryName, hardlink.NewLink)
			// A Link directly to linkPathname will fail if it exists, so do a
			// Link+Rename using a temporary filename.
			if err := fsutil.ForceLink(targetPathname, tmpName); err != nil {
				t.lastError = err
				t.logger.Println(err)
				continue
			}
			if err := fsutil.ForceRename(tmpName, linkPathname); err != nil {
				t.logger.Println(err)
				if err := fsutil.ForceRemove(tmpName); err != nil {
					t.lastError = err
					t.logger.Println(err)
				}
			} else {
				t.logger.Printf("Linked: %s => %s\n",
					linkPathname, targetPathname)
			}
		}
	}
}

func (t *uType) doDeletes(pathsToDelete []string, triggers *triggers.Triggers,
	takeAction bool) {
	for _, pathname := range pathsToDelete {
		fullPathname := path.Join(t.rootDirectoryName, pathname)
		triggers.Match(pathname)
		if takeAction {
			if err := fsutil.ForceRemoveAll(fullPathname); err != nil {
				t.lastError = err
				t.logger.Println(err)
			} else {
				t.logger.Printf("Deleted: %s\n", fullPathname)
			}
		}
	}
}

func (t *uType) makeDirectories(directoriesToMake []sub.Inode,
	triggers *triggers.Triggers, takeAction bool) {
	for _, newdir := range directoriesToMake {
		if t.skipPath(newdir.Name) {
			continue
		}
		fullPathname := path.Join(t.rootDirectoryName, newdir.Name)
		triggers.Match(newdir.Name)
		if takeAction {
			inode, ok := newdir.GenericInode.(*filesystem.DirectoryInode)
			if !ok {
				t.logger.Println("%s is not a directory!\n", newdir.Name)
				continue
			}
			if err := inode.Write(fullPathname); err != nil {
				t.lastError = err
				t.logger.Println(err)
			} else {
				t.logger.Printf("Made directory: %s (mode=%s)\n",
					fullPathname, inode.Mode)
			}
		}
	}
}

func (t *uType) changeInodes(inodesToChange []sub.Inode,
	triggers *triggers.Triggers, takeAction bool) {
	for _, inode := range inodesToChange {
		fullPathname := path.Join(t.rootDirectoryName, inode.Name)
		triggers.Match(inode.Name)
		if takeAction {
			if err := filesystem.ForceWriteMetadata(inode,
				fullPathname); err != nil {
				t.lastError = err
				t.logger.Println(err)
				continue
			}
			t.logger.Printf("Changed inode: %s\n", fullPathname)
		}
	}
}

func (t *uType) skipPath(pathname string) bool {
	if t.skipFilter.Match(pathname) {
		return true
	}
	if pathname == "/.subd" {
		return true
	}
	if strings.HasPrefix(pathname, "/.subd/") {
		return true
	}
	return false
}
