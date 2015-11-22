package herd

import (
	"flag"
	"github.com/Symantec/Dominator/lib/image"
	"log"
	"runtime"
	"time"
)

var (
	maxConnAttempts = flag.Uint("maxConnAttempts", 1000,
		"Maximum number of concurrent connection attempts")
)

func newHerd(imageServerAddress string, logger *log.Logger) *Herd {
	var herd Herd
	herd.imageServerAddress = imageServerAddress
	herd.logger = logger
	herd.subsByName = make(map[string]*Sub)
	herd.imagesByName = make(map[string]*image.Image)
	herd.makeConnectionSemaphore = make(chan bool, *maxConnAttempts)
	herd.pollSemaphore = make(chan bool, runtime.NumCPU()*2)
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
		herd.previousScanDuration = time.Since(herd.currentScanStartTime)
		return true
	}
	if herd.nextSubToPoll == 0 {
		herd.currentScanStartTime = time.Now()
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

func (herd *Herd) countSelectedSubs(selectFunc func(*Sub) bool) uint64 {
	herd.RLock()
	defer herd.RUnlock()
	if selectFunc == nil {
		return uint64(len(herd.subsByIndex))
	}
	count := 0
	for _, sub := range herd.subsByIndex {
		if selectFunc(sub) {
			count++
		}
	}
	return uint64(count)
}

func (herd *Herd) getSelectedSubs(selectFunc func(*Sub) bool) []*Sub {
	herd.RLock()
	defer herd.RUnlock()
	subs := make([]*Sub, 0, len(herd.subsByIndex))
	for _, sub := range herd.subsByIndex {
		if selectFunc == nil || selectFunc(sub) {
			subs = append(subs, sub)
		}
	}
	return subs
}
