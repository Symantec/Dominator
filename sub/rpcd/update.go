package rpcd

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	jsonlib "github.com/Symantec/Dominator/lib/json"
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

func (t *rpcType) Update(conn *srpc.Conn, request sub.UpdateRequest,
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
	t.lastUpdateError = nil
	if request.Wait {
		return t.doUpdate(request, fs.RootDirectoryName())
	}
	go t.doUpdate(request, fs.RootDirectoryName())
	return nil
}

func (t *rpcType) doUpdate(request sub.UpdateRequest,
	rootDirectoryName string) error {
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
		t.makeHardlinks(request.HardlinksToMake, rootDirectoryName,
			&oldTriggers, "", false)
		t.doDeletes(request.PathsToDelete, rootDirectoryName, &oldTriggers,
			false)
		t.changeInodes(request.InodesToChange, rootDirectoryName, &oldTriggers,
			false)
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
	t.makeHardlinks(request.HardlinksToMake, rootDirectoryName,
		request.Triggers, t.objectsDir, true)
	t.doDeletes(request.PathsToDelete, rootDirectoryName, request.Triggers,
		true)
	t.changeInodes(request.InodesToChange, rootDirectoryName, request.Triggers,
		true)
	fsChangeDuration := time.Since(fsChangeStartTime)
	matchedNewTriggers := request.Triggers.GetMatchedTriggers()
	file, err = os.Create(t.oldTriggersFilename)
	if err == nil {
		writer := bufio.NewWriter(file)
		if err := jsonlib.WriteWithIndent(writer, "    ",
			request.Triggers.Triggers); err != nil {
			t.logger.Printf("Error marshaling triggers: %s", err)
		}
		writer.Flush()
		file.Close()
	}
	if runTriggers(matchedNewTriggers, "start", t.logger) {
		t.lastUpdateHadTriggerFailures = true
	}
	timeTaken := time.Since(startTime)
	if t.lastUpdateError != nil {
		t.logger.Printf("Update(): last error: %s\n", t.lastUpdateError)
	} else {
		t.rwLock.Lock()
		t.lastSuccessfulImageName = request.ImageName
		t.rwLock.Unlock()
	}
	t.logger.Printf("Update() completed in %s (change window: %s)\n",
		timeTaken, fsChangeDuration)
	return t.lastUpdateError
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
			t.lastUpdateError = err
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
				t.lastUpdateError = err
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
				t.lastUpdateError = err
			}
		}
	}
}

func makeRegularInode(fullPathname string,
	inode *filesystem.RegularInode, multiplyUsedObjects map[hash.Hash]uint64,
	objectsDir string, logger *log.Logger) error {
	var objectPathname string
	if inode.Size > 0 {
		objectPathname = path.Join(objectsDir,
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
	inode *filesystem.SymlinkInode, logger *log.Logger) error {
	if err := inode.Write(fullPathname); err != nil {
		logger.Println(err)
		return err
	}
	logger.Printf("Made symlink inode: %s -> %s\n", fullPathname, inode.Symlink)
	return nil
}

func makeSpecialInode(fullPathname string, inode *filesystem.SpecialInode,
	logger *log.Logger) error {
	if err := inode.Write(fullPathname); err != nil {
		logger.Println(err)
		return err
	}
	logger.Printf("Made special inode: %s\n", fullPathname)
	return nil
}

func (t *rpcType) makeHardlinks(hardlinksToMake []sub.Hardlink,
	rootDirectoryName string, triggers *triggers.Triggers, tmpDir string,
	takeAction bool) {
	tmpName := path.Join(tmpDir, "temporaryHardlink")
	for _, hardlink := range hardlinksToMake {
		triggers.Match(hardlink.NewLink)
		if takeAction {
			targetPathname := path.Join(rootDirectoryName, hardlink.Target)
			linkPathname := path.Join(rootDirectoryName, hardlink.NewLink)
			// A Link directly to linkPathname will fail if it exists, so do a
			// Link+Rename using a temporary filename.
			if err := fsutil.ForceLink(targetPathname, tmpName); err != nil {
				t.lastUpdateError = err
				t.logger.Println(err)
				continue
			}
			if err := fsutil.ForceRename(tmpName, linkPathname); err != nil {
				t.logger.Println(err)
				if err := fsutil.ForceRemove(tmpName); err != nil {
					t.lastUpdateError = err
					t.logger.Println(err)
				}
			} else {
				t.logger.Printf("Linked: %s => %s\n",
					linkPathname, targetPathname)
			}
		}
	}
}

func (t *rpcType) doDeletes(pathsToDelete []string, rootDirectoryName string,
	triggers *triggers.Triggers, takeAction bool) {
	for _, pathname := range pathsToDelete {
		fullPathname := path.Join(rootDirectoryName, pathname)
		triggers.Match(pathname)
		if takeAction {
			if err := fsutil.ForceRemoveAll(fullPathname); err != nil {
				t.lastUpdateError = err
				t.logger.Println(err)
			} else {
				t.logger.Printf("Deleted: %s\n", fullPathname)
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
				t.lastUpdateError = err
				t.logger.Println(err)
			} else {
				t.logger.Printf("Made directory: %s (mode=%s)\n",
					fullPathname, inode.Mode)
			}
		}
	}
}

func (t *rpcType) changeInodes(inodesToChange []sub.Inode,
	rootDirectoryName string, triggers *triggers.Triggers, takeAction bool) {
	for _, inode := range inodesToChange {
		fullPathname := path.Join(rootDirectoryName, inode.Name)
		triggers.Match(inode.Name)
		if takeAction {
			if err := filesystem.ForceWriteMetadata(inode,
				fullPathname); err != nil {
				t.lastUpdateError = err
				t.logger.Println(err)
				continue
			}
			t.logger.Printf("Changed inode: %s\n", fullPathname)
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
				if !runCommand(logger, "reboot") {
					hadFailures = true
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
		if !runCommand(logger,
			"run-in-mntns", ppid, "service", trigger.Service, action) {
			hadFailures = true
		}
	}
	if needRestart {
		logger.Printf("%sAction: service subd restart\n", logPrefix)
		if !runCommand(logger,
			"run-in-mntns", ppid, "service", "subd", "restart") {
			hadFailures = true
		}
	}
	return hadFailures
}

func runCommand(logger *log.Logger, name string, args ...string) bool {
	cmd := exec.Command(name, args...)
	if logs, err := cmd.CombinedOutput(); err != nil {
		errMsg := "error running: " + name
		for _, arg := range args {
			errMsg += " " + arg
		}
		errMsg += ": " + err.Error()
		logger.Printf("error running: %s\n", errMsg)
		logger.Println(string(logs))
		return false
	}
	return true
}
