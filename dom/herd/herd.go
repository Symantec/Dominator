package herd

import (
	"github.com/Symantec/Dominator/lib/image"
	"log"
	"runtime"
	"syscall"
	"time"
)

func newHerd(imageServerAddress string, logger *log.Logger) *Herd {
	var herd Herd
	herd.imageServerAddress = imageServerAddress
	herd.logger = logger
	herd.subsByName = make(map[string]*Sub)
	herd.imagesByName = make(map[string]*image.Image)
	// Limit concurrent connection attempts so that the file descriptor limit is
	// not exceeded.
	var rlim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim); err != nil {
		panic(err)
	}
	maxConnAttempts := rlim.Cur - 50
	maxConnAttempts = (maxConnAttempts / 100)
	if maxConnAttempts < 1 {
		maxConnAttempts = 1
	} else {
		maxConnAttempts *= 100
	}
	herd.connectionSemaphore = make(chan bool, maxConnAttempts)
	herd.pollSemaphore = make(chan bool, runtime.NumCPU()*2)
	herd.currentScanStartTime = time.Now()
	return &herd
}

func (herd *Herd) decrementConnectionSemaphore() {
	<-herd.connectionSemaphore
}

func (herd *Herd) waitForCompletion() {
	for count := 0; count < cap(herd.connectionSemaphore); count++ {
		herd.connectionSemaphore <- true
	}
	for count := 0; count < cap(herd.connectionSemaphore); count++ {
		<-herd.connectionSemaphore
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
	herd.connectionSemaphore <- true
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
