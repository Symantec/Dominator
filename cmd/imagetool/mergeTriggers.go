package main

import (
	"fmt"
	"os"

	json "github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/triggers"
)

func mergeTriggersSubcommand(args []string) {
	err := mergeTriggers(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error merging triggers: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
