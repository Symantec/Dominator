package triggers

import (
	"regexp"
)

type Trigger struct {
	MatchLines   []string
	matchRegexes []*regexp.Regexp
	Service      string
	HighImpact   bool
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

func (triggers *Triggers) Match(line string) {
	triggers.match(line)
}

func (triggers *Triggers) GetMatchedTriggers() []*Trigger {
	return triggers.getMatchedTriggers()
}
