package triggers

import (
	"regexp"
)

type MergeableTriggers struct {
	triggers map[string]*mergeableTrigger // Key: service name.
}

type mergeableTrigger struct {
	matchLines map[string]struct{}
	doReboot   bool
	highImpact bool
}

type Trigger struct {
	MatchLines   []string
	matchRegexes []*regexp.Regexp
	Service      string
	DoReboot     bool
	HighImpact   bool
}

func (trigger *Trigger) ReplaceStrings(replaceFunc func(string) string) {
	trigger.replaceStrings(replaceFunc)
}

type Triggers struct {
	Triggers          []*Trigger
	compiled          bool
	matchedTriggers   map[*Trigger]bool
	unmatchedTriggers map[*Trigger]bool
}

func Decode(jsonData []byte) (*Triggers, error) {
	return decode(jsonData)
}

func Load(filename string) (*Triggers, error) {
	return load(filename)
}

func New() *Triggers {
	return newTriggers()
}

func (triggers *Triggers) ReplaceStrings(replaceFunc func(string) string) {
	triggers.replaceStrings(replaceFunc)
}

func (mt *MergeableTriggers) ExportTriggers() *Triggers {
	return mt.exportTriggers()
}

func (mt *MergeableTriggers) Merge(triggers *Triggers) {
	mt.merge(triggers)
}

func (triggers *Triggers) Match(line string) {
	triggers.match(line)
}

func (triggers *Triggers) GetMatchedTriggers() []*Trigger {
	return triggers.getMatchedTriggers()
}
