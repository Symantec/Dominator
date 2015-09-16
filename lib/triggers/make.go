package triggers

import (
	"regexp"
)

func (triggers *Triggers) addTrigger(matchLines []string, actionLine string) {
	var trigger Trigger
	trigger.MatchLines = matchLines
	trigger.ActionLine = actionLine
	*triggers = append(*triggers, &trigger)
}

func (triggers Triggers) compile() error {
	for _, trigger := range triggers {
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
