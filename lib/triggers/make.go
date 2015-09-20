package triggers

import (
	"regexp"
)

func newTriggers() *Triggers {
	return &Triggers{}
}

func (triggers *Triggers) compile() error {
	if triggers.compiled {
		return nil
	}
	for _, trigger := range triggers.Triggers {
		trigger.matchRegexes = make([]*regexp.Regexp, len(trigger.MatchLines))
		for index, line := range trigger.MatchLines {
			var err error
			trigger.matchRegexes[index], err = regexp.Compile("^" + line)
			if err != nil {
				return err
			}
		}
	}
	triggers.compiled = true
	return nil
}
