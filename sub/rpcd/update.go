package rpcd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/triggers"
	"github.com/Symantec/Dominator/proto/sub"
	"os"
	"path"
	"strings"
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
	var oldTriggers *triggers.Triggers
	file, err := os.Open(oldTriggersFilename)
	if err == nil {
		decoder := json.NewDecoder(file)
		var trig triggers.Triggers
		err = decoder.Decode(&trig.Triggers)
		file.Close()
		if err == nil {
			oldTriggers = &trig
		} else {
			logger.Printf("Error decoding old triggers: %s", err.Error())
		}
	}
	if oldTriggers != nil {
		// TODO(rgooch): Process old triggers before making any changes.
		matchedOldTriggers := oldTriggers.GetMatchedTriggers()
		_ = matchedOldTriggers
	}
	processDeletes(request, rootDirectoryName)
	processMakeDirectories(request, rootDirectoryName)
	matchedNewTriggers := request.Triggers.GetMatchedTriggers()
	// TODO(rgooch): Remove debugging output.
	if len(matchedNewTriggers) > 0 {
		fmt.Println("Triggers:")
		b, _ := json.Marshal(matchedNewTriggers)
		var out bytes.Buffer
		json.Indent(&out, b, "", "    ")
		out.WriteTo(os.Stdout)
	}
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
	// TODO(rgooch): Remove debugging hack and implement.
	time.Sleep(time.Second * 15)
	logger.Printf("Update() complete\n")
}

func clearUpdateInProgress() {
	rwLock.Lock()
	defer rwLock.Unlock()
	updateInProgress = false
}

func processDeletes(request sub.UpdateRequest, rootDirectoryName string) {
	for _, pathname := range request.PathsToDelete {
		fullPathname := path.Join(rootDirectoryName, pathname)
		// TODO(rgooch): Remove debugging.
		fmt.Printf("Delete: %s\n", fullPathname)
		request.Triggers.Match(pathname)
		// TODO(rgooch): Implement.
	}
}

func processMakeDirectories(request sub.UpdateRequest,
	rootDirectoryName string) {
	for _, newdir := range request.DirectoriesToMake {
		if scannerConfiguration.ScanFilter.Match(newdir.Name) {
			continue
		}
		if newdir.Name == "/.subd" {
			continue
		}
		if strings.HasPrefix(newdir.Name, "/.subd/") {
			continue
		}
		fullPathname := path.Join(rootDirectoryName, newdir.Name)
		// TODO(rgooch): Remove debugging.
		fmt.Printf("Mkdir: %s\n", fullPathname)
		// TODO(rgooch): Implement.
		request.Triggers.Match(newdir.Name)
	}
}
