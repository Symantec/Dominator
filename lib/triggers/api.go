package triggers

import (
	"regexp"
)

type Trigger struct {
	MatchLines   []string
	matchRegexes []*regexp.Regexp
	Command      string
	HighImpact   bool
}

type Triggers struct {
	Triggers          []*Trigger
	matchedTriggers   map[*Trigger]bool
	unmatchedTriggers map[*Trigger]bool
}

func New() *Triggers {
	return newTriggers()
}

func (triggers *Triggers) AddTrigger(matchLines []string, command string,
	highImpact bool) {
	triggers.addTrigger(matchLines, command, highImpact)
}

func (triggers *Triggers) Match(line string) {
	triggers.match(line)
}

func (triggers *Triggers) GetMatchedTriggers() []*Trigger {
	return triggers.getMatchedTriggers()
}
