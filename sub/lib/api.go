package lib

import (
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/filter"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/triggers"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
)

type TriggersRunner func(triggers []*triggers.Trigger, action string,
	logger log.Logger) bool

type uType struct {
	rootDirectoryName  string
	objectsDir         string
	skipFilter         *filter.Filter
	runTriggers        TriggersRunner
	disableTriggers    bool
	logger             log.Logger
	lastError          error
	hadTriggerFailures bool
	fsChangeDuration   time.Duration
}

func Update(request sub.UpdateRequest, rootDirectoryName string,
	objectsDir string, oldTriggers *triggers.Triggers,
	skipFilter *filter.Filter, triggersRunner TriggersRunner,
	logger log.Logger) (
	bool, time.Duration, error) {
	if skipFilter == nil {
		skipFilter = new(filter.Filter)
	}
	updateObj := &uType{
		rootDirectoryName: rootDirectoryName,
		objectsDir:        objectsDir,
		skipFilter:        skipFilter,
		runTriggers:       triggersRunner,
		logger:            logger,
	}
	err := updateObj.update(request, oldTriggers)
	return updateObj.hadTriggerFailures, updateObj.fsChangeDuration, err
}
