package herd

import (
	"github.com/Symantec/Dominator/dom/mdb"
)

func (herd *Herd) mdbUpdate(mdb *mdb.Mdb) {
	herd.waitForCompletion()
	herd.subsByIndex = nil
	if herd.subsByName == nil {
		herd.subsByName = make(map[string]*Sub)
	}
	for _, sub := range herd.subsByName {
		sub.hostname = "" // Flag sub as potentially not in the new MDB.
	}
	for _, machine := range mdb.Machines {
		sub := herd.subsByName[machine.Hostname]
		if sub == nil {
			sub = new(Sub)
			sub.herd = herd
			herd.subsByName[machine.Hostname] = sub
		}
		sub.hostname = machine.Hostname // Flag sub as being in the new MDB.
		sub.requiredImage = machine.RequiredImage
		sub.plannedImage = machine.PlannedImage
	}
	// Delete unflagged subs (those not in the new MDB).
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
	imageUseMap := make(map[string]bool) // Unreferenced by default.
	for _, sub := range herd.subsByName {
		herd.subsByIndex = append(herd.subsByIndex, sub)
		imageUseMap[sub.requiredImage] = true
		imageUseMap[sub.plannedImage] = true
	}
	// Clean up unreferenced images.
	for name, used := range imageUseMap {
		if !used {
			delete(herd.imagesByName, name)
		}
	}
}
