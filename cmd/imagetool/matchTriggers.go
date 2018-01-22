package main

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/triggers"
)

var quitEarly = errors.New("quit early")

func matchTriggersSubcommand(args []string) {
	if err := matchTriggers(args[0], args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error matching triggers: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
