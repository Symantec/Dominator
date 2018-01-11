package triggers

import (
	"sort"
)

func (mt *MergeableTriggers) exportTriggers() *Triggers {
	if len(mt.triggers) < 1 {
		return nil
	}
	triggerList := make([]*Trigger, 0, len(mt.triggers))
	serviceNames := make([]string, 0, len(mt.triggers))
	for service := range mt.triggers {
		serviceNames = append(serviceNames, service)
	}
	sort.Strings(serviceNames)
	for _, service := range serviceNames {
		trigger := mt.triggers[service]
		matchLines := make([]string, 0, len(trigger.matchLines))
		for matchLine := range trigger.matchLines {
			matchLines = append(matchLines, matchLine)
		}
		sort.Strings(matchLines)
		triggerList = append(triggerList, &Trigger{
			MatchLines: matchLines,
			Service:    service,
			DoReboot:   trigger.doReboot,
			HighImpact: trigger.highImpact,
		})
	}
	triggers := New()
	triggers.Triggers = triggerList
	return triggers
}

func (mt *MergeableTriggers) merge(triggers *Triggers) {
	if triggers == nil || len(triggers.Triggers) < 1 {
		return
	}
	if mt.triggers == nil {
		mt.triggers = make(map[string]*mergeableTrigger, len(triggers.Triggers))
	}
	for _, trigger := range triggers.Triggers {
		trig := mt.triggers[trigger.Service]
		if trig == nil {
			trig = new(mergeableTrigger)
			trig.matchLines = make(map[string]struct{})
			mt.triggers[trigger.Service] = trig
		}
		for _, matchLine := range trigger.MatchLines {
			trig.matchLines[matchLine] = struct{}{}
		}
		if trigger.DoReboot {
			trig.doReboot = true
		}
		if trigger.HighImpact {
			trig.highImpact = true
		}
	}
}
