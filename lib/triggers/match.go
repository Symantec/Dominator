package triggers

func (triggers *Triggers) match(line string) {
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
	mtriggers := make([]*Trigger, 0, len(triggers.matchedTriggers))
	for trigger, _ := range triggers.matchedTriggers {
		mtriggers = append(mtriggers, trigger)
	}
	triggers.matchedTriggers = nil
	triggers.unmatchedTriggers = nil
	return mtriggers
}
