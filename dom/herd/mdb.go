package herd

import (
	"github.com/Symantec/Dominator/dom/mdb"
)

func (herd *Herd) mdbUpdate(mdb *mdb.Mdb) {
	herd.subsByIndex = nil
	if herd.subsByName == nil {
		herd.subsByName = make(map[string]*Sub)
	}
	for _, sub := range herd.subsByName {
		sub.hostname = ""
	}
	for _, machine := range mdb.Machines {
		sub := herd.subsByName[machine.Hostname]
		if sub == nil {
			sub = new(Sub)
			herd.subsByName[machine.Hostname] = sub
		}
		sub.hostname = machine.Hostname
		sub.requiredImage = machine.RequiredImage
		sub.plannedImage = machine.PlannedImage
	}
	subsToDelete := make([]string, 0)
	for hostname, sub := range herd.subsByName {
		if sub.hostname == "" {
			subsToDelete = append(subsToDelete, hostname)
		}
	}
	for _, hostname := range subsToDelete {
		delete(herd.subsByName, hostname)
	}
	herd.subsByIndex = make([]*Sub, 0, len(herd.subsByName))
	for _, sub := range herd.subsByName {
		herd.subsByIndex = append(herd.subsByIndex, sub)
	}
}
