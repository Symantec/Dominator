package fleet

import (
	"github.com/Symantec/Dominator/dom/mdb"
)

func (fleet *Fleet) mdbUpdate(mdb *mdb.Mdb) {
	fleet.nextSubToPoll = 0
	fleet.subs = make([]*Sub, 0, mdb.Len())
	for _, machine := range mdb.Machines {
		var sub Sub
		sub.hostname = machine.Hostname
		sub.requiredImage = machine.RequiredImage
		sub.plannedImage = machine.PlannedImage
		fleet.subs = append(fleet.subs, &sub)
	}
}
