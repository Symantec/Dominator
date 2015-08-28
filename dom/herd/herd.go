package herd

import (
	"runtime"
	"time"
)

func newHerd(imageServerAddress string) *Herd {
	var herd Herd
	herd.imageServerAddress = imageServerAddress
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
