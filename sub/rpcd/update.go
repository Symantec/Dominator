package rpcd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/triggers"
	"github.com/Symantec/Dominator/proto/sub"
	"os"
	"os/exec"
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
	if len(oldTriggers.Triggers) > 0 {
		processDeletes(request, rootDirectoryName, &oldTriggers, false)
		processMakeDirectories(request, rootDirectoryName, &oldTriggers, false)
		matchedOldTriggers := oldTriggers.GetMatchedTriggers()
		runTriggers(matchedOldTriggers, "stop")
	}
	processDeletes(request, rootDirectoryName, request.Triggers, true)
	processMakeDirectories(request, rootDirectoryName, request.Triggers, true)
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
	// TODO(rgooch): Remove debugging hack and implement.
	time.Sleep(time.Second * 15)
	logger.Printf("Update() complete\n")
}

func clearUpdateInProgress() {
	rwLock.Lock()
	defer rwLock.Unlock()
	updateInProgress = false
}

func processDeletes(request sub.UpdateRequest, rootDirectoryName string,
	triggers *triggers.Triggers, takeAction bool) {
	for _, pathname := range request.PathsToDelete {
		fullPathname := path.Join(rootDirectoryName, pathname)
		triggers.Match(pathname)
		if takeAction {
			// TODO(rgooch): Remove debugging.
			fmt.Printf("Delete: %s\n", fullPathname)
			// TODO(rgooch): Implement.
		}
	}
}

func processMakeDirectories(request sub.UpdateRequest, rootDirectoryName string,
	triggers *triggers.Triggers, takeAction bool) {
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
		triggers.Match(newdir.Name)
		if takeAction {
			// TODO(rgooch): Remove debugging.
			fmt.Printf("Mkdir: %s\n", fullPathname)
			// TODO(rgooch): Implement.
		}
	}
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
				err := cmd.Run()
				if err != nil {
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
		err := cmd.Run()
		if err != nil {
			logger.Print(err)
		}
		// TODO(rgooch): Implement.
	}
}
