package herd

import (
	"github.com/Symantec/Dominator/dom/mdb"
)

func (herd *Herd) mdbUpdate(mdb *mdb.Mdb) {
	herd.nextSubToPoll = 0
	herd.subs = make([]*Sub, 0, mdb.Len())
	for _, machine := range mdb.Machines {
		var sub Sub
		sub.hostname = machine.Hostname
		sub.requiredImage = machine.RequiredImage
		sub.plannedImage = machine.PlannedImage
		herd.subs = append(herd.subs, &sub)
	}
}
