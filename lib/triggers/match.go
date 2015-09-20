package triggers

func (triggers *Triggers) match(line string) {
	triggers.compile()
	if triggers.matchedTriggers == nil {
		triggers.matchedTriggers = make(map[*Trigger]bool)
		triggers.unmatchedTriggers = make(map[*Trigger]bool)
		for _, trigger := range triggers.Triggers {
			triggers.unmatchedTriggers[trigger] = true
		}
	}
	for trigger, _ := range triggers.unmatchedTriggers {
		for _, regex := range trigger.matchRegexes {
			if regex.MatchString(line) {
				triggers.matchedTriggers[trigger] = true
				delete(triggers.unmatchedTriggers, trigger)
				break
			}
		}
	}
}

func (triggers *Triggers) getMatchedTriggers() []*Trigger {
	mTriggers := make([]*Trigger, 0, len(triggers.matchedTriggers))
	for trigger, _ := range triggers.matchedTriggers {
		mTriggers = append(mTriggers, trigger)
	}
	triggers.matchedTriggers = nil
	triggers.unmatchedTriggers = nil
	return mTriggers
}
