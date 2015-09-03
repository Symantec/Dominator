package herd

import (
	"github.com/Symantec/Dominator/lib/image"
	"log"
	"runtime"
	"time"
)

func newHerd(imageServerAddress string, logger *log.Logger) *Herd {
	var herd Herd
	herd.imageServerAddress = imageServerAddress
	herd.logger = logger
	herd.subsByName = make(map[string]*Sub)
	herd.imagesByName = make(map[string]*image.Image)
	herd.makeConnectionSemaphore = make(chan bool, 1000)
	herd.pollSemaphore = make(chan bool, runtime.NumCPU()*2)
	herd.previousScanStartTime = time.Now()
	herd.currentScanStartTime = time.Now()
	return &herd
}

func (herd *Herd) decrementConnectionSemaphore() {
	<-herd.makeConnectionSemaphore
}

func (herd *Herd) waitForCompletion() {
	for count := 0; count < cap(herd.makeConnectionSemaphore); count++ {
		herd.makeConnectionSemaphore <- true
	}
	for count := 0; count < cap(herd.makeConnectionSemaphore); count++ {
		<-herd.makeConnectionSemaphore
	}
}

func (herd *Herd) pollNextSub() bool {
	if herd.nextSubToPoll >= uint(len(herd.subsByIndex)) {
		herd.nextSubToPoll = 0
		herd.previousScanStartTime = herd.currentScanStartTime
		herd.currentScanStartTime = time.Now()
		return true
	}
	sub := herd.subsByIndex[herd.nextSubToPoll]
	herd.nextSubToPoll++
	if sub.busy { // Quick lockless check.
		return false
	}
	herd.makeConnectionSemaphore <- true
	go func() {
		defer herd.decrementConnectionSemaphore()
		if !sub.tryMakeBusy() {
			return
		}
		sub.connectAndPoll()
		sub.makeUnbusy()
	}()
	return false
}

func (herd *Herd) getSelectedSubs(selectFunc func(*Sub) bool) []*Sub {
	httpdHerd.RLock()
	defer httpdHerd.RUnlock()
	subs := make([]*Sub, 0, len(httpdHerd.subsByIndex))
	for _, sub := range httpdHerd.subsByIndex {
		if selectFunc == nil || selectFunc(sub) {
			subs = append(subs, sub)
		}
	}
	return subs
}
