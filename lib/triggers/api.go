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

type Triggers []*Trigger

func New() *Triggers {
	return &Triggers{}
}

func (triggers *Triggers) AddTrigger(matchLines []string, command string,
	highImpact bool) {
	triggers.addTrigger(matchLines, command, highImpact)
}
