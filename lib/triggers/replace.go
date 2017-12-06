package triggers

func (trigger *Trigger) replaceStrings(replaceFunc func(string) string) {
	for index, str := range trigger.MatchLines {
		trigger.MatchLines[index] = replaceFunc(str)
	}
	trigger.Service = replaceFunc(trigger.Service)
}

func (triggers *Triggers) replaceStrings(replaceFunc func(string) string) {
	if triggers != nil {
		for _, trigger := range triggers.Triggers {
			trigger.ReplaceStrings(replaceFunc)
		}
	}
}
