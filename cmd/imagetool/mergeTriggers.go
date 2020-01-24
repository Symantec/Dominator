package main

import (
	"fmt"
	"os"

	json "github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/triggers"
)

func mergeTriggersSubcommand(args []string, logger log.DebugLogger) error {
	if err := mergeTriggers(args); err != nil {
		return fmt.Errorf("Error merging triggers: %s", err)
	}
	return nil
}

func mergeTriggers(triggerFiles []string) error {
	mergeableTriggers := &triggers.MergeableTriggers{}
	for _, triggerFile := range triggerFiles {
		trig, err := triggers.Load(triggerFile)
		if err != nil {
			return err
		}
		mergeableTriggers.Merge(trig)
	}
	trig := mergeableTriggers.ExportTriggers()
	return json.WriteWithIndent(os.Stdout, "    ", trig.Triggers)
}
