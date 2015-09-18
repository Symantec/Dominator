package triggers

import (
	"regexp"
)

func newTriggers() *Triggers {
	return &Triggers{}
}

func (triggers *Triggers) addTrigger(matchLines []string, command string,
	highImpact bool) {
	var trigger Trigger
	trigger.MatchLines = matchLines
	trigger.Command = command
	trigger.HighImpact = highImpact
	triggers.Triggers = append(triggers.Triggers, &trigger)
}

func (triggers Triggers) compile() error {
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
	return nil
}
