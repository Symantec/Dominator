package rpcd

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/triggers"
	"github.com/Symantec/Dominator/proto/sub"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"
)

var (
	readOnly = flag.Bool("readOnly", false,
		"If true, refuse all Fetch and Update requests. For debugging only")
	disableUpdates = flag.Bool("disableUpdates", false,
		"If true, refuse all Update requests. For debugging only")
	disableTriggers = flag.Bool("disableTriggers", false,
		"If true, do not run any triggers. For debugging only")
)

func (t *rpcType) Update(request sub.UpdateRequest,
	reply *sub.UpdateResponse) error {
	if *readOnly || *disableUpdates {
		txt := "Update() rejected due to read-only mode"
		logger.Println(txt)
		return errors.New(txt)
	}
	rwLock.Lock()
	defer rwLock.Unlock()
	fs := fileSystemHistory.FileSystem()
	if fs == nil {
		return errors.New("No file-system history yet")
	}
	logger.Printf("Update()\n")
	if fetchInProgress {
		logger.Println("Error: fetch already in progress")
		return errors.New("fetch already in progress")
	}
	if updateInProgress {
		logger.Println("Error: update progress")
		return errors.New("update in progress")
	}
	updateInProgress = true
	go doUpdate(request, fs.RootDirectoryName())
	return nil
}

func doUpdate(request sub.UpdateRequest, rootDirectoryName string) {
	defer clearUpdateInProgress()
	startTime := time.Now()
	var oldTriggers triggers.Triggers
	file, err := os.Open(oldTriggersFilename)
	if err == nil {
		decoder := json.NewDecoder(file)
		var trig triggers.Triggers
		err = decoder.Decode(&trig.Triggers)
		file.Close()
		if err == nil {
			oldTriggers = trig
		} else {
			logger.Printf("Error decoding old triggers: %s", err.Error())
		}
	}
	copyFilesToCache(request.FilesToCopyToCache, rootDirectoryName)
	makeObjectCopies(request.MultiplyUsedObjects)
	if len(oldTriggers.Triggers) > 0 {
		makeInodes(request.InodesToMake, rootDirectoryName,
			request.MultiplyUsedObjects, &oldTriggers, false)
		makeHardlinks(request.HardlinksToMake, rootDirectoryName,
			&oldTriggers, false)
		doDeletes(request.PathsToDelete, rootDirectoryName, &oldTriggers, false)
		makeDirectories(request.DirectoriesToMake, rootDirectoryName,
			&oldTriggers, false)
		changeInodes(request.InodesToChange, rootDirectoryName, &oldTriggers,
			false)
		matchedOldTriggers := oldTriggers.GetMatchedTriggers()
		runTriggers(matchedOldTriggers, "stop")
	}
	makeInodes(request.InodesToMake, rootDirectoryName,
		request.MultiplyUsedObjects, request.Triggers, true)
	makeHardlinks(request.HardlinksToMake, rootDirectoryName,
		request.Triggers, true)
	doDeletes(request.PathsToDelete, rootDirectoryName, request.Triggers, true)
	makeDirectories(request.DirectoriesToMake, rootDirectoryName,
		request.Triggers, true)
	changeInodes(request.InodesToChange, rootDirectoryName, request.Triggers,
		true)
	matchedNewTriggers := request.Triggers.GetMatchedTriggers()
	file, err = os.Create(oldTriggersFilename)
	if err == nil {
		b, err := json.Marshal(request.Triggers.Triggers)
		if err == nil {
			var out bytes.Buffer
			json.Indent(&out, b, "", "    ")
			out.WriteTo(file)
		} else {
			logger.Printf("Error marshaling triggers: %s", err.Error())
		}
		file.Close()
	}
	runTriggers(matchedNewTriggers, "start")
	timeTaken := time.Since(startTime)
	logger.Printf("Update() completed in %s\n", timeTaken)
	// TODO(rgooch): Remove debugging hack and implement.
	time.Sleep(time.Second * 15)
	logger.Printf("Post-Update() debugging sleep complete\n")
}

func clearUpdateInProgress() {
	rwLock.Lock()
	defer rwLock.Unlock()
	updateInProgress = false
}

func copyFilesToCache(filesToCopyToCache []sub.FileToCopyToCache,
	rootDirectoryName string) {
	for _, fileToCopy := range filesToCopyToCache {
		sourcePathname := path.Join(rootDirectoryName, fileToCopy.Name)
		destPathname := path.Join(objectsDir,
			objectcache.HashToFilename(fileToCopy.Hash))
		if copyFile(destPathname, sourcePathname) {
			logger.Printf("Copied: %s to cache\n", sourcePathname)
		}
	}
}

func copyFile(destPathname, sourcePathname string) bool {
	sourceFile, err := os.Open(sourcePathname)
	if err != nil {
		logger.Println(err)
		return false
	}
	defer sourceFile.Close()
	dirname := path.Dir(destPathname)
	if err := os.MkdirAll(dirname, syscall.S_IRWXU); err != nil {
		return false
	}
	destFile, err := os.Create(destPathname)
	if err != nil {
		logger.Println(err)
		return false
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return false
	}
	return true
}

func makeObjectCopies(multiplyUsedObjects map[hash.Hash]uint64) {
	for hash, numCopies := range multiplyUsedObjects {
		if numCopies < 2 {
			continue
		}
		objectPathname := path.Join(objectsDir,
			objectcache.HashToFilename(hash))
		for numCopies--; numCopies > 0; numCopies-- {
			ext := strings.Repeat("~", int(numCopies))
			if copyFile(objectPathname+ext, objectPathname) {
				logger.Printf("Copied object: %x%s\n", hash, ext)
			}
		}
	}
}

func makeInodes(inodesToMake []sub.Inode, rootDirectoryName string,
	multiplyUsedObjects map[hash.Hash]uint64, triggers *triggers.Triggers,
	takeAction bool) {
	for _, inode := range inodesToMake {
		fullPathname := path.Join(rootDirectoryName, inode.Name)
		triggers.Match(inode.Name)
		if takeAction {
			switch inode := inode.GenericInode.(type) {
			case *filesystem.RegularInode:
				makeRegularInode(fullPathname, inode, multiplyUsedObjects)
			case *filesystem.SymlinkInode:
				makeSymlinkInode(fullPathname, inode)
			case *filesystem.SpecialInode:
				makeSpecialInode(fullPathname, inode)
			}
		}
	}
}

func makeRegularInode(fullPathname string, inode *filesystem.RegularInode,
	multiplyUsedObjects map[hash.Hash]uint64) {
	var err error
	if inode.Size > 0 {
		objectPathname := path.Join(objectsDir,
			objectcache.HashToFilename(inode.Hash))
		numCopies := multiplyUsedObjects[inode.Hash]
		if numCopies > 1 {
			numCopies--
			objectPathname += strings.Repeat("~", int(numCopies))
			if numCopies < 2 {
				delete(multiplyUsedObjects, inode.Hash)
			}
		}
		err = os.Rename(objectPathname, fullPathname)
	} else {
		_, err = os.Create(fullPathname)
	}
	if err != nil {
		logger.Println(err)
		return
	}
	if err := inode.WriteMetadata(fullPathname); err != nil {
		logger.Println(err)
	} else {
		if inode.Size > 0 {
			logger.Printf("Made inode: %s from: %x\n", fullPathname, inode.Hash)
		} else {
			logger.Printf("Made empty inode: %s\n", fullPathname)
		}
	}
}

func makeSymlinkInode(fullPathname string, inode *filesystem.SymlinkInode) {
	if err := inode.Write(fullPathname); err != nil {
		logger.Println(err)
	} else {
		logger.Printf("Made symlink inode: %s -> %s\n",
			fullPathname, inode.Symlink)
	}
}

func makeSpecialInode(fullPathname string, inode *filesystem.SpecialInode) {
	if err := inode.Write(fullPathname); err != nil {
		logger.Println(err)
	} else {
		logger.Printf("Made special inode: %s\n", fullPathname)
	}
}

func makeHardlinks(hardlinksToMake []sub.Hardlink, rootDirectoryName string,
	triggers *triggers.Triggers, takeAction bool) {
	for _, hardlink := range hardlinksToMake {
		triggers.Match(hardlink.NewLink)
		if takeAction {
			targetPathname := path.Join(rootDirectoryName, hardlink.Target)
			linkPathname := path.Join(rootDirectoryName, hardlink.NewLink)
			if err := os.Link(targetPathname, linkPathname); err != nil {
				logger.Println(err)
			} else {
				logger.Printf("Linked: %s => %s\n",
					linkPathname, targetPathname)
			}
		}
	}
}

func doDeletes(pathsToDelete []string, rootDirectoryName string,
	triggers *triggers.Triggers, takeAction bool) {
	for _, pathname := range pathsToDelete {
		fullPathname := path.Join(rootDirectoryName, pathname)
		triggers.Match(pathname)
		if takeAction {
			if err := os.RemoveAll(fullPathname); err != nil {
				logger.Println(err)
			} else {
				logger.Printf("Deleted: %s\n", fullPathname)
			}
		}
	}
}

func makeDirectories(directoriesToMake []sub.Inode, rootDirectoryName string,
	triggers *triggers.Triggers, takeAction bool) {
	for _, newdir := range directoriesToMake {
		if skipPath(newdir.Name) {
			continue
		}
		fullPathname := path.Join(rootDirectoryName, newdir.Name)
		triggers.Match(newdir.Name)
		if takeAction {
			inode, ok := newdir.GenericInode.(*filesystem.DirectoryInode)
			if !ok {
				logger.Println("%s is not a directory!\n", newdir.Name)
				continue
			}
			if err := inode.Write(fullPathname); err != nil {
				logger.Println(err)
			} else {
				logger.Printf("Made directory: %s\n", fullPathname)
			}
		}
	}
}

func changeInodes(inodesToChange []sub.Inode, rootDirectoryName string,
	triggers *triggers.Triggers, takeAction bool) {
	for _, inode := range inodesToChange {
		fullPathname := path.Join(rootDirectoryName, inode.Name)
		triggers.Match(inode.Name)
		if takeAction {
			if err := inode.WriteMetadata(fullPathname); err != nil {
				logger.Println(err)
				continue
			}
			logger.Printf("Changed inode: %s\n", fullPathname)
		}
	}
}

func skipPath(pathname string) bool {
	if scannerConfiguration.ScanFilter.Match(pathname) {
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

func runTriggers(triggers []*triggers.Trigger, action string) {
	logPrefix := ""
	if *disableTriggers {
		logPrefix = "Disabled: "
	}
	// For "start" action, if there is a reboot trigger, just do that one.
	if action == "start" {
		for _, trigger := range triggers {
			if trigger.Service == "reboot" {
				logger.Print(logPrefix, "Rebooting")
				if *disableTriggers {
					return
				}
				cmd := exec.Command("reboot")
				cmd.Stdout = os.Stdout
				if err := cmd.Run(); err != nil {
					logger.Print(err)
				}
				return
			}
		}
	}
	ppid := fmt.Sprint(os.Getppid())
	for _, trigger := range triggers {
		if trigger.Service == "reboot" && action == "stop" {
			continue
		}
		logger.Printf("%sAction: service %s %s\n",
			logPrefix, trigger.Service, action)
		if *disableTriggers {
			continue
		}
		cmd := exec.Command("run-in-mntns", ppid, "service", action,
			trigger.Service)
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			logger.Print(err)
		}
		// TODO(rgooch): Implement.
	}
}
