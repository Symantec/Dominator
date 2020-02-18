package triggers

import (
	"io"
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
	DoReboot     bool `json:",omitempty"`
	HighImpact   bool `json:",omitempty"`
}

func (trigger *Trigger) ReplaceStrings(replaceFunc func(string) string) {
	trigger.replaceStrings(replaceFunc)
}

type Triggers struct {
	Triggers          []*Trigger
	compiled          bool
	matchedTriggers   map[*Trigger]struct{}
	unmatchedTriggers map[*Trigger]struct{}
}

func Decode(jsonData []byte) (*Triggers, error) {
	return decode(jsonData)
}

func Load(filename string) (*Triggers, error) {
	return load(filename)
}

func Read(reader io.Reader) (*Triggers, error) {
	return read(reader)
}

func New() *Triggers {
	return newTriggers()
}

func (triggers *Triggers) Len() int {
	return len(triggers.Triggers)
}

func (triggers *Triggers) Less(left, right int) bool {
	return triggers.Triggers[left].Service < triggers.Triggers[right].Service
}

func (triggers *Triggers) ReplaceStrings(replaceFunc func(string) string) {
	triggers.replaceStrings(replaceFunc)
}

func (triggers *Triggers) Swap(left, right int) {
	triggers.Triggers[left], triggers.Triggers[right] =
		triggers.Triggers[right], triggers.Triggers[left]
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

func (triggers *Triggers) GetMatchStatistics() (nMatched, nUnmatched uint) {
	return triggers.getMatchStatistics()
}
