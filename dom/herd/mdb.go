package herd

import (
	"github.com/Symantec/Dominator/lib/mdb"
	"time"
)

func (herd *Herd) mdbUpdate(mdb *mdb.Mdb) {
	numNew, numDeleted := herd.mdbUpdateNoLogging(mdb)
	pluralNew := "s"
	if numNew == 1 {
		pluralNew = ""
	}
	pluralDeleted := "s"
	if numDeleted == 1 {
		pluralDeleted = ""
	}
	herd.logger.Printf("MDB update: %d new sub%s, %d removed sub%s",
		numNew, pluralNew, numDeleted, pluralDeleted)
}

func (herd *Herd) mdbUpdateNoLogging(mdb *mdb.Mdb) (int, int) {
	herd.Lock()
	defer herd.Unlock()
	startTime := time.Now()
	numNew := 0
	numDeleted := 0
	herd.subsByIndex = make([]*Sub, 0, len(mdb.Machines))
	// Mark for delete all current subs, then later unmark ones in the new MDB.
	subsToDelete := make(map[string]bool)
	for _, sub := range herd.subsByName {
		subsToDelete[sub.hostname] = true
	}
	for _, machine := range mdb.Machines { // Sorted by Hostname.
		sub := herd.subsByName[machine.Hostname]
		if sub == nil {
			sub = new(Sub)
			sub.herd = herd
			sub.hostname = machine.Hostname
			herd.subsByName[machine.Hostname] = sub
			numNew++
		}
		subsToDelete[sub.hostname] = false
		herd.subsByIndex = append(herd.subsByIndex, sub)
		if sub.requiredImage != machine.RequiredImage ||
			sub.plannedImage != machine.PlannedImage {
			sub.generationCount = 0 // Force a full poll.
		}
		sub.requiredImage = machine.RequiredImage
		sub.plannedImage = machine.PlannedImage
		herd.getImageHaveLock(sub.requiredImage) // Preload.
		if herd.getImageHaveLock(sub.plannedImage) == nil {
			sub.havePlannedImage = false
		}
	}
	// Delete flagged subs (those not in the new MDB).
	for subHostname, toDelete := range subsToDelete {
		if toDelete {
			delete(herd.subsByName, subHostname)
			numDeleted++
		}
	}
	imageUseMap := make(map[string]bool) // Unreferenced by default.
	for _, sub := range herd.subsByName {
		imageUseMap[sub.requiredImage] = true
		imageUseMap[sub.plannedImage] = true
	}
	// Clean up unreferenced images.
	for name, used := range imageUseMap {
		if !used {
			delete(herd.imagesByName, name)
		}
	}
	mdbUpdateTimeDistribution.Add(time.Since(startTime))
	return numNew, numDeleted
}
