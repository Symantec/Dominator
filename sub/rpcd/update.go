package rpcd

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	jsonlib "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/triggers"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/lib"
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
	if err := t.getUpdateLock(); err != nil {
		t.logger.Println(err)
		return err
	}
	t.logger.Printf("Update()\n")
	fs := t.fileSystemHistory.FileSystem()
	if request.Wait {
		return t.updateAndUnlock(request, fs.RootDirectoryName())
	}
	go t.updateAndUnlock(request, fs.RootDirectoryName())
	return nil
}

func (t *rpcType) getUpdateLock() error {
	if *readOnly || *disableUpdates {
		return errors.New("Update() rejected due to read-only mode")
	}
	fs := t.fileSystemHistory.FileSystem()
	if fs == nil {
		return errors.New("No file-system history yet")
	}
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	if t.fetchInProgress {
		return errors.New("Fetch() in progress")
	}
	if t.updateInProgress {
		return errors.New("Update() already in progress")
	}
	t.updateInProgress = true
	t.lastUpdateError = nil
	return nil
}

func (t *rpcType) updateAndUnlock(request sub.UpdateRequest,
	rootDirectoryName string) error {
	defer t.clearUpdateInProgress()
	defer t.scannerConfiguration.BoostCpuLimit(t.logger)
	t.disableScannerFunc(true)
	defer t.disableScannerFunc(false)
	startTime := time.Now()
	oldTriggers := &triggers.MergeableTriggers{}
	file, err := os.Open(t.oldTriggersFilename)
	if err == nil {
		decoder := json.NewDecoder(file)
		var trig triggers.Triggers
		err = decoder.Decode(&trig.Triggers)
		file.Close()
		if err == nil {
			oldTriggers.Merge(&trig)
		} else {
			t.logger.Printf("Error decoding old triggers: %s", err.Error())
		}
	}
	if request.Triggers != nil {
		// Merge new triggers into old triggers. This supports initial
		// Domination of a machine and when the old triggers are incomplete.
		oldTriggers.Merge(request.Triggers)
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
	}
	hadTriggerFailures, fsChangeDuration, lastUpdateError := lib.Update(
		request, rootDirectoryName, t.objectsDir, oldTriggers.ExportTriggers(),
		t.scannerConfiguration.ScanFilter, runTriggers, t.logger)
	t.lastUpdateHadTriggerFailures = hadTriggerFailures
	t.lastUpdateError = lastUpdateError
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

// Returns true if there were failures.
func runTriggers(triggers []*triggers.Trigger, action string,
	logger log.Logger) bool {
	doReboot := false
	hadFailures := false
	needRestart := false
	logPrefix := ""
	if *disableTriggers {
		logPrefix = "Disabled: "
	}
	ppid := fmt.Sprint(os.Getppid())
	for _, trigger := range triggers {
		if trigger.DoReboot && action == "start" {
			doReboot = true
			break
		}
	}
	for _, trigger := range triggers {
		if trigger.Service == "subd" {
			// Never kill myself, just restart.
			if action == "start" {
				needRestart = true
			}
			continue
		}
		if trigger.Service == "" {
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
			if trigger.DoReboot && action == "start" {
				doReboot = false
			}
		}
	}
	if doReboot {
		logger.Print(logPrefix, "Rebooting")
		if *disableTriggers {
			return hadFailures
		}
		if !runCommand(logger, "reboot") {
			hadFailures = true
		}
		return hadFailures
	} else if needRestart {
		logger.Printf("%sAction: service subd restart\n", logPrefix)
		if !runCommand(logger,
			"run-in-mntns", ppid, "service", "subd", "restart") {
			hadFailures = true
		}
	}
	return hadFailures
}

// Returns true on success, else false.
func runCommand(logger log.Logger, name string, args ...string) bool {
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
