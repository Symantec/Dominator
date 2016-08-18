package herd

import (
	filegenclient "github.com/Symantec/Dominator/lib/filegen/client"
	"github.com/Symantec/Dominator/lib/mdb"
	"time"
)

func (herd *Herd) mdbUpdate(mdb *mdb.Mdb) {
	numNew, numDeleted, numChanged := herd.mdbUpdateNoLogging(mdb)
	pluralNew := "s"
	if numNew == 1 {
		pluralNew = ""
	}
	pluralDeleted := "s"
	if numDeleted == 1 {
		pluralDeleted = ""
	}
	pluralChanged := "s"
	if numChanged == 1 {
		pluralChanged = ""
	}
	herd.logger.Printf(
		"MDB update: %d new sub%s, %d removed sub%s, %d changed sub%s",
		numNew, pluralNew, numDeleted, pluralDeleted, numChanged, pluralChanged)
}

func (herd *Herd) mdbUpdateNoLogging(mdb *mdb.Mdb) (int, int, int) {
	herd.Lock()
	defer herd.Unlock()
	startTime := time.Now()
	numNew := 0
	numDeleted := 0
	numChanged := 0
	herd.subsByIndex = make([]*Sub, 0, len(mdb.Machines))
	// Mark for delete all current subs, then later unmark ones in the new MDB.
	subsToDelete := make(map[string]struct{})
	for _, sub := range herd.subsByName {
		subsToDelete[sub.mdb.Hostname] = struct{}{}
	}
	for _, machine := range mdb.Machines { // Sorted by Hostname.
		sub := herd.subsByName[machine.Hostname]
		img, _ := herd.getImageHaveLock(machine.RequiredImage) // Preload.
		if sub == nil {
			sub = new(Sub)
			sub.herd = herd
			sub.mdb = machine
			herd.subsByName[machine.Hostname] = sub
			sub.fileUpdateChannel = herd.computedFilesManager.Add(
				filegenclient.Machine{machine, sub.getComputedFiles(img)}, 16)
			numNew++
		} else {
			if sub.mdb.RequiredImage != machine.RequiredImage {
				sub.computedInodes = nil
			}
			if sub.mdb != machine {
				sub.mdb = machine
				sub.generationCount = 0 // Force a full poll.
				herd.computedFilesManager.Update(
					filegenclient.Machine{machine, sub.getComputedFiles(img)})
				numChanged++
			}
		}
		delete(subsToDelete, machine.Hostname)
		herd.subsByIndex = append(herd.subsByIndex, sub)
		if img, _ = herd.getImageHaveLock(machine.PlannedImage); img == nil {
			sub.havePlannedImage = false
		} else {
			sub.havePlannedImage = true
		}
	}
	// Delete flagged subs (those not in the new MDB).
	for subHostname := range subsToDelete {
		herd.computedFilesManager.Remove(subHostname)
		delete(herd.subsByName, subHostname)
		numDeleted++
	}
	unusedImages := make(map[string]struct{})
	for name := range herd.imagesByName {
		unusedImages[name] = struct{}{}
	}
	for _, sub := range herd.subsByName {
		delete(unusedImages, sub.mdb.RequiredImage)
		delete(unusedImages, sub.mdb.PlannedImage)
	}
	// Clean up unreferenced images.
	for name := range unusedImages {
		delete(herd.imagesByName, name)
	}
	mdbUpdateTimeDistribution.Add(time.Since(startTime))
	return numNew, numDeleted, numChanged
}
