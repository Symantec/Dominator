package rpcd

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/triggers"
	"github.com/Symantec/Dominator/proto/sub"
	"io"
	"log"
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

func (t *rpcType) Update(conn *srpc.Conn) error {
	var request sub.UpdateRequest
	var response sub.UpdateResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if err := t.update(request, &response); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response)
}

func (t *rpcType) update(request sub.UpdateRequest,
	reply *sub.UpdateResponse) error {
	if *readOnly || *disableUpdates {
		txt := "Update() rejected due to read-only mode"
		t.logger.Println(txt)
		return errors.New(txt)
	}
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	fs := t.fileSystemHistory.FileSystem()
	if fs == nil {
		return errors.New("No file-system history yet")
	}
	t.logger.Printf("Update()\n")
	if t.fetchInProgress {
		t.logger.Println("Error: fetch already in progress")
		return errors.New("fetch already in progress")
	}
	if t.updateInProgress {
		t.logger.Println("Error: update progress")
		return errors.New("update in progress")
	}
	t.updateInProgress = true
	go t.doUpdate(request, fs.RootDirectoryName())
	return nil
}

func (t *rpcType) doUpdate(request sub.UpdateRequest,
	rootDirectoryName string) {
	defer t.clearUpdateInProgress()
	t.disableScannerFunc(true)
	defer t.disableScannerFunc(false)
	startTime := time.Now()
	var oldTriggers triggers.Triggers
	file, err := os.Open(t.oldTriggersFilename)
	if err == nil {
		decoder := json.NewDecoder(file)
		var trig triggers.Triggers
		err = decoder.Decode(&trig.Triggers)
		file.Close()
		if err == nil {
			oldTriggers = trig
		} else {
			t.logger.Printf("Error decoding old triggers: %s", err.Error())
		}
	}
	t.copyFilesToCache(request.FilesToCopyToCache, rootDirectoryName)
	t.makeObjectCopies(request.MultiplyUsedObjects)
	t.lastUpdateHadTriggerFailures = false
	if len(oldTriggers.Triggers) > 0 {
		t.makeDirectories(request.DirectoriesToMake, rootDirectoryName,
			&oldTriggers, false)
		t.makeInodes(request.InodesToMake, rootDirectoryName,
			request.MultiplyUsedObjects, &oldTriggers, false)
		makeHardlinks(request.HardlinksToMake, rootDirectoryName,
			&oldTriggers, "", false, t.logger)
		doDeletes(request.PathsToDelete, rootDirectoryName, &oldTriggers, false,
			t.logger)
		changeInodes(request.InodesToChange, rootDirectoryName, &oldTriggers,
			false, t.logger)
		matchedOldTriggers := oldTriggers.GetMatchedTriggers()
		if runTriggers(matchedOldTriggers, "stop", t.logger) {
			t.lastUpdateHadTriggerFailures = true
		}
	}
	fsChangeStartTime := time.Now()
	t.makeDirectories(request.DirectoriesToMake, rootDirectoryName,
		request.Triggers, true)
	t.makeInodes(request.InodesToMake, rootDirectoryName,
		request.MultiplyUsedObjects, request.Triggers, true)
	makeHardlinks(request.HardlinksToMake, rootDirectoryName,
		request.Triggers, t.objectsDir, true, t.logger)
	doDeletes(request.PathsToDelete, rootDirectoryName, request.Triggers, true,
		t.logger)
	changeInodes(request.InodesToChange, rootDirectoryName, request.Triggers,
		true, t.logger)
	fsChangeDuration := time.Since(fsChangeStartTime)
	matchedNewTriggers := request.Triggers.GetMatchedTriggers()
	file, err = os.Create(t.oldTriggersFilename)
	if err == nil {
		b, err := json.Marshal(request.Triggers.Triggers)
		if err == nil {
			var out bytes.Buffer
			json.Indent(&out, b, "", "    ")
			out.WriteTo(file)
		} else {
			t.logger.Printf("Error marshaling triggers: %s", err.Error())
		}
		file.Close()
	}
	if runTriggers(matchedNewTriggers, "start", t.logger) {
		t.lastUpdateHadTriggerFailures = true
	}
	timeTaken := time.Since(startTime)
	t.logger.Printf("Update() completed in %s (change window: %s)\n",
		timeTaken, fsChangeDuration)
}

func (t *rpcType) clearUpdateInProgress() {
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	t.updateInProgress = false
}

func (t *rpcType) copyFilesToCache(filesToCopyToCache []sub.FileToCopyToCache,
	rootDirectoryName string) {
	for _, fileToCopy := range filesToCopyToCache {
		sourcePathname := path.Join(rootDirectoryName, fileToCopy.Name)
		destPathname := path.Join(t.objectsDir,
			objectcache.HashToFilename(fileToCopy.Hash))
		if err := copyFile(destPathname, sourcePathname); err != nil {
			t.logger.Println(err)
		} else {
			t.logger.Printf("Copied: %s to cache\n", sourcePathname)
		}
	}
}

func copyFile(destPathname, sourcePathname string) error {
	sourceFile, err := os.Open(sourcePathname)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	dirname := path.Dir(destPathname)
	if err := os.MkdirAll(dirname, syscall.S_IRWXU); err != nil {
		return err
	}
	destFile, err := os.Create(destPathname)
	if err != nil {
		return err
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (t *rpcType) makeObjectCopies(multiplyUsedObjects map[hash.Hash]uint64) {
	for hash, numCopies := range multiplyUsedObjects {
		if numCopies < 2 {
			continue
		}
		objectPathname := path.Join(t.objectsDir,
			objectcache.HashToFilename(hash))
		for numCopies--; numCopies > 0; numCopies-- {
			ext := strings.Repeat("~", int(numCopies))
			if err := copyFile(objectPathname+ext, objectPathname); err != nil {
				t.logger.Println(err)
			} else {
				t.logger.Printf("Copied object: %x%s\n", hash, ext)
			}
		}
	}
}

func (t *rpcType) makeInodes(inodesToMake []sub.Inode, rootDirectoryName string,
	multiplyUsedObjects map[hash.Hash]uint64, triggers *triggers.Triggers,
	takeAction bool) {
	for _, inode := range inodesToMake {
		fullPathname := path.Join(rootDirectoryName, inode.Name)
		triggers.Match(inode.Name)
		if takeAction {
			switch inode := inode.GenericInode.(type) {
			case *filesystem.RegularInode:
				makeRegularInode(fullPathname, inode, multiplyUsedObjects,
					t.objectsDir, t.logger)
			case *filesystem.SymlinkInode:
				makeSymlinkInode(fullPathname, inode, t.logger)
			case *filesystem.SpecialInode:
				makeSpecialInode(fullPathname, inode, t.logger)
			}
		}
	}
}

func makeRegularInode(fullPathname string,
	inode *filesystem.RegularInode, multiplyUsedObjects map[hash.Hash]uint64,
	objectsDir string, logger *log.Logger) {
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
			} else {
				multiplyUsedObjects[inode.Hash] = numCopies
			}
		}
		err = fsutil.ForceRename(objectPathname, fullPathname)
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
			logger.Printf("Made inode: %s from: %x\n",
				fullPathname, inode.Hash)
		} else {
			logger.Printf("Made empty inode: %s\n", fullPathname)
		}
	}
}

func makeSymlinkInode(fullPathname string,
	inode *filesystem.SymlinkInode, logger *log.Logger) {
	if err := inode.Write(fullPathname); err != nil {
		logger.Println(err)
	} else {
		logger.Printf("Made symlink inode: %s -> %s\n",
			fullPathname, inode.Symlink)
	}
}

func makeSpecialInode(fullPathname string, inode *filesystem.SpecialInode,
	logger *log.Logger) {
	if err := inode.Write(fullPathname); err != nil {
		logger.Println(err)
	} else {
		logger.Printf("Made special inode: %s\n", fullPathname)
	}
}

func makeHardlinks(hardlinksToMake []sub.Hardlink, rootDirectoryName string,
	triggers *triggers.Triggers, tmpDir string, takeAction bool,
	logger *log.Logger) {
	tmpName := path.Join(tmpDir, "temporaryHardlink")
	for _, hardlink := range hardlinksToMake {
		triggers.Match(hardlink.NewLink)
		if takeAction {
			targetPathname := path.Join(rootDirectoryName, hardlink.Target)
			linkPathname := path.Join(rootDirectoryName, hardlink.NewLink)
			// A Link directly to linkPathname will fail if it exists, so do a
			// Link+Rename using a temporary filename.
			if err := fsutil.ForceLink(targetPathname, tmpName); err != nil {
				logger.Println(err)
				continue
			}
			if err := fsutil.ForceRename(tmpName, linkPathname); err != nil {
				logger.Println(err)
				if err := fsutil.ForceRemove(tmpName); err != nil {
					logger.Println(err)
				}
			} else {
				logger.Printf("Linked: %s => %s\n",
					linkPathname, targetPathname)
			}
		}
	}
}

func doDeletes(pathsToDelete []string, rootDirectoryName string,
	triggers *triggers.Triggers, takeAction bool, logger *log.Logger) {
	for _, pathname := range pathsToDelete {
		fullPathname := path.Join(rootDirectoryName, pathname)
		triggers.Match(pathname)
		if takeAction {
			if err := fsutil.ForceRemoveAll(fullPathname); err != nil {
				logger.Println(err)
			} else {
				logger.Printf("Deleted: %s\n", fullPathname)
			}
		}
	}
}

func (t *rpcType) makeDirectories(directoriesToMake []sub.Inode,
	rootDirectoryName string, triggers *triggers.Triggers, takeAction bool) {
	for _, newdir := range directoriesToMake {
		if t.skipPath(newdir.Name) {
			continue
		}
		fullPathname := path.Join(rootDirectoryName, newdir.Name)
		triggers.Match(newdir.Name)
		if takeAction {
			inode, ok := newdir.GenericInode.(*filesystem.DirectoryInode)
			if !ok {
				t.logger.Println("%s is not a directory!\n", newdir.Name)
				continue
			}
			if err := inode.Write(fullPathname); err != nil {
				t.logger.Println(err)
			} else {
				t.logger.Printf("Made directory: %s\n", fullPathname)
			}
		}
	}
}

func changeInodes(inodesToChange []sub.Inode, rootDirectoryName string,
	triggers *triggers.Triggers, takeAction bool, logger *log.Logger) {
	for _, inode := range inodesToChange {
		fullPathname := path.Join(rootDirectoryName, inode.Name)
		triggers.Match(inode.Name)
		if takeAction {
			if err := filesystem.ForceWriteMetadata(inode,
				fullPathname); err != nil {
				logger.Println(err)
				continue
			}
			logger.Printf("Changed inode: %s\n", fullPathname)
		}
	}
}

func (t *rpcType) skipPath(pathname string) bool {
	if t.scannerConfiguration.ScanFilter.Match(pathname) {
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

func runTriggers(triggers []*triggers.Trigger, action string,
	logger *log.Logger) bool {
	hadFailures := false
	needRestart := false
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
					return hadFailures
				}
				err := runCommand("reboot")
				if err != nil {
					hadFailures = true
					logger.Print(err)
				}
				return hadFailures
			}
		}
	}
	ppid := fmt.Sprint(os.Getppid())
	for _, trigger := range triggers {
		if trigger.Service == "reboot" && action == "stop" {
			continue
		}
		if trigger.Service == "subd" {
			// Never kill myself, just restart.
			if action == "start" {
				needRestart = true
			}
			continue
		}
		logger.Printf("%sAction: service %s %s\n",
			logPrefix, trigger.Service, action)
		if *disableTriggers {
			continue
		}
		err := runCommand("run-in-mntns", ppid, "service", trigger.Service,
			action)
		if err != nil {
			hadFailures = true
			logger.Print(err)
		}
	}
	if needRestart {
		logger.Printf("%sAction: service subd restart\n", logPrefix)
		err := runCommand("run-in-mntns", ppid, "service", "subd", "restart")
		if err != nil {
			hadFailures = true
			logger.Print(err)
		}
	}
	return hadFailures
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errMsg := "error running: " + name
		for _, arg := range args {
			errMsg += " " + arg
		}
		errMsg += ": " + err.Error()
		return errors.New(errMsg)
	}
	return nil
}
