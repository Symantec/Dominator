package triggers

func (triggers *Triggers) match(line string) {
	triggers.compile()
	if triggers.matchedTriggers == nil {
		triggers.matchedTriggers = make(map[*Trigger]struct{})
		triggers.unmatchedTriggers = make(map[*Trigger]struct{},
			len(triggers.Triggers))
		for _, trigger := range triggers.Triggers {
			triggers.unmatchedTriggers[trigger] = struct{}{}
		}
	}
	for trigger := range triggers.unmatchedTriggers {
		for _, regex := range trigger.matchRegexes {
			if regex.MatchString(line) {
				triggers.matchedTriggers[trigger] = struct{}{}
				delete(triggers.unmatchedTriggers, trigger)
				break
			}
		}
	}
}

func (triggers *Triggers) getMatchedTriggers() []*Trigger {
	mTriggers := make([]*Trigger, 0, len(triggers.matchedTriggers))
	for trigger := range triggers.matchedTriggers {
		mTriggers = append(mTriggers, trigger)
	}
	triggers.matchedTriggers = nil
	triggers.unmatchedTriggers = nil
	return mTriggers
}

func (triggers *Triggers) getMatchStatistics() (nMatched, nUnmatched uint) {
	if triggers.matchedTriggers == nil {
		nUnmatched = uint(len(triggers.Triggers))
	} else {
		nUnmatched = uint(len(triggers.unmatchedTriggers))
	}
	return uint(len(triggers.matchedTriggers)), nUnmatched
}
