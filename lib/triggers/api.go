package triggers

import (
	"regexp"
)

type Trigger struct {
	MatchLines   []string
	matchRegexes []*regexp.Regexp
	ActionLine   string
}

type Triggers []*Trigger

func New() *Triggers {
	return &Triggers{}
}

func (triggers Triggers) AddTrigger(matchLines []string, actionLine string) {
	triggers.addTrigger(matchLines, actionLine)
}
