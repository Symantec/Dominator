package herd

import (
	filegenclient "github.com/Symantec/Dominator/lib/filegen/client"
	"github.com/Symantec/Dominator/lib/mdb"
	"reflect"
	"time"
)

func (herd *Herd) mdbUpdate(mdb *mdb.Mdb) {
	numNew, numDeleted, numChanged, wantedImages := herd.mdbUpdateGetLock(mdb)
	// Clean up unreferenced images.
	herd.imageManager.SetImageInterestList(wantedImages, true)
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

func (herd *Herd) mdbUpdateGetLock(mdb *mdb.Mdb) (
	int, int, int, map[string]struct{}) {
	herd.LockWithTimeout(time.Minute)
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
	wantedImages := make(map[string]struct{})
	wantedImages[herd.defaultImageName] = struct{}{}
	wantedImages[herd.nextDefaultImageName] = struct{}{}
	for _, machine := range mdb.Machines { // Sorted by Hostname.
		if machine.Hostname == "" {
			herd.logger.Printf("Empty Hostname field, ignoring \"%s\"\n",
				machine)
			continue
		}
		sub := herd.subsByName[machine.Hostname]
		wantedImages[machine.RequiredImage] = struct{}{}
		wantedImages[machine.PlannedImage] = struct{}{}
		img := herd.imageManager.GetNoError(machine.RequiredImage)
		if sub == nil {
			sub = &Sub{
				herd:          herd,
				mdb:           machine,
				cancelChannel: make(chan struct{}),
			}
			herd.subsByName[machine.Hostname] = sub
			sub.fileUpdateChannel = herd.computedFilesManager.Add(
				filegenclient.Machine{machine, sub.getComputedFiles(img)}, 16)
			numNew++
		} else {
			if sub.mdb.RequiredImage != machine.RequiredImage {
				if sub.status == statusSynced {
					sub.status = statusWaitingToPoll
				}
			}
			if !reflect.DeepEqual(sub.mdb, machine) {
				sub.mdb = machine
				sub.generationCount = 0 // Force a full poll.
				herd.computedFilesManager.Update(
					filegenclient.Machine{machine, sub.getComputedFiles(img)})
				sub.sendCancel()
				numChanged++
			}
		}
		delete(subsToDelete, machine.Hostname)
		herd.subsByIndex = append(herd.subsByIndex, sub)
		img = herd.imageManager.GetNoError(machine.PlannedImage)
		if img == nil {
			sub.havePlannedImage = false
		} else {
			sub.havePlannedImage = true
		}
	}
	delete(wantedImages, "")
	// Delete flagged subs (those not in the new MDB).
	for subHostname := range subsToDelete {
		sub := herd.subsByName[subHostname]
		sub.busyFlagMutex.Lock()
		sub.deleting = true
		if sub.clientResource != nil {
			sub.clientResource.ScheduleClose()
		}
		sub.busyFlagMutex.Unlock()
		herd.computedFilesManager.Remove(subHostname)
		delete(herd.subsByName, subHostname)
		numDeleted++
	}
	mdbUpdateTimeDistribution.Add(time.Since(startTime))
	return numNew, numDeleted, numChanged, wantedImages
}
