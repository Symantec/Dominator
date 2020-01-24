package main

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/triggers"
)

var quitEarly = errors.New("quit early")

func matchTriggersSubcommand(args []string, logger log.DebugLogger) error {
	if err := matchTriggers(args[0], args[1]); err != nil {
		return fmt.Errorf("Error matching triggers: %s", err)
	}
	return nil
}

func matchTriggers(image string, triggersFile string) error {
	fs, err := getTypedImage(image)
	if err != nil {
		return err
	}
	trig, err := triggers.Load(triggersFile)
	if err != nil {
		return err
	}
	fs.ForEachFile(
		func(name string, inodeNumber uint64,
			inode filesystem.GenericInode) error {
			if _, nUnmatched := trig.GetMatchStatistics(); nUnmatched < 1 {
				return quitEarly
			}
			trig.Match(name)
			return nil
		})
	trig = &triggers.Triggers{Triggers: trig.GetMatchedTriggers()}
	sort.Sort(trig)
	return json.WriteWithIndent(os.Stdout, "    ", trig.Triggers)
}
