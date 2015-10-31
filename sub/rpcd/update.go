package rpcd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

func (t *rpcType) Update(request sub.UpdateRequest,
	reply *sub.UpdateResponse) error {
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
	processFilesToCopyToCache(request.FilesToCopyToCache, rootDirectoryName)
	if len(oldTriggers.Triggers) > 0 {
		processMakeInodes(request.InodesToMake, rootDirectoryName,
			request.MultiplyUsedObjects, &oldTriggers, false)
		processHardlinksToMake(request.HardlinksToMake, rootDirectoryName,
			&oldTriggers, false)
		processDeletes(request.PathsToDelete, rootDirectoryName, &oldTriggers,
			false)
		processMakeDirectories(request.DirectoriesToMake, rootDirectoryName,
			&oldTriggers, false)
		processChangeDirectories(request.DirectoriesToChange, rootDirectoryName,
			&oldTriggers, false)
		processChangeInodes(request.InodesToChange, rootDirectoryName,
			&oldTriggers, false)
		matchedOldTriggers := oldTriggers.GetMatchedTriggers()
		runTriggers(matchedOldTriggers, "stop")
	}
	processMakeInodes(request.InodesToMake, rootDirectoryName,
		request.MultiplyUsedObjects, request.Triggers, true)
	processHardlinksToMake(request.HardlinksToMake, rootDirectoryName,
		request.Triggers, true)
	processDeletes(request.PathsToDelete, rootDirectoryName, request.Triggers,
		true)
	processMakeDirectories(request.DirectoriesToMake, rootDirectoryName,
		request.Triggers, true)
	processChangeDirectories(request.DirectoriesToChange, rootDirectoryName,
		request.Triggers, true)
	processChangeInodes(request.InodesToChange, rootDirectoryName,
		request.Triggers, true)
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

func processFilesToCopyToCache(filesToCopyToCache []sub.FileToCopyToCache,
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

func processMakeInodes(inodesToMake []sub.Inode, rootDirectoryName string,
	multiplyUsedObjects map[hash.Hash]uint64, triggers *triggers.Triggers,
	takeAction bool) {
	for _, inode := range inodesToMake {
		fullPathname := path.Join(rootDirectoryName, inode.Name)
		triggers.Match(inode.Name)
		if takeAction {
			logger.Printf("Make inode: %s\n", fullPathname)
			// TODO(rgooch): Implement.
		}
	}
}

func processHardlinksToMake(hardlinksToMake []sub.Hardlink,
	rootDirectoryName string, triggers *triggers.Triggers, takeAction bool) {
	for _, hardlink := range hardlinksToMake {
		triggers.Match(hardlink.NewLink)
		if takeAction {
			targetPathname := path.Join(rootDirectoryName, hardlink.Target)
			linkPathname := path.Join(rootDirectoryName, hardlink.NewLink)
			err := os.Link(targetPathname, linkPathname)
			if err != nil {
				logger.Println(err)
			} else {
				logger.Printf("Linked: %s => %s\n",
					linkPathname, targetPathname)
			}
		}
	}
}

func processDeletes(pathsToDelete []string, rootDirectoryName string,
	triggers *triggers.Triggers, takeAction bool) {
	for _, pathname := range pathsToDelete {
		fullPathname := path.Join(rootDirectoryName, pathname)
		triggers.Match(pathname)
		if takeAction {
			err := os.RemoveAll(fullPathname)
			if err != nil {
				logger.Println(err)
			} else {
				logger.Printf("Deleted: %s\n", fullPathname)
			}
		}
	}
}

func processMakeDirectories(directoriesToMake []sub.Directory,
	rootDirectoryName string, triggers *triggers.Triggers, takeAction bool) {
	for _, newdir := range directoriesToMake {
		if skipPath(newdir.Name) {
			continue
		}
		fullPathname := path.Join(rootDirectoryName, newdir.Name)
		triggers.Match(newdir.Name)
		if takeAction {
			logger.Printf("Mkdir: %s\n", fullPathname)
			// TODO(rgooch): Implement.
		}
	}
}

func processChangeDirectories(directoriesToChange []sub.Directory,
	rootDirectoryName string, triggers *triggers.Triggers, takeAction bool) {
	for _, directory := range directoriesToChange {
		if takeAction {
			logger.Printf("Change directory: %s\n", directory.Name)
			// TODO(rgooch): Implement.
		}
	}
}

func processChangeInodes(inodesToChange []sub.Inode,
	rootDirectoryName string, triggers *triggers.Triggers, takeAction bool) {
	for _, inode := range inodesToChange {
		if takeAction {
			logger.Printf("Change inode: %s\n", inode.Name)
			// TODO(rgooch): Implement.
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
	// For "start" action, if there is a reboot trigger, just do that one.
	if action == "start" {
		for _, trigger := range triggers {
			if trigger.Service == "reboot" {
				logger.Print("Rebooting")
				// TODO(rgooch): Remove debugging output.
				cmd := exec.Command("echo", "reboot")
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
		logger.Printf("Action: service %s %s\n", trigger.Service, action)
		// TODO(rgooch): Remove debugging output.
		cmd := exec.Command("run-in-mntns", ppid, "echo", "service", action,
			trigger.Service)
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			logger.Print(err)
		}
		// TODO(rgooch): Implement.
	}
}
