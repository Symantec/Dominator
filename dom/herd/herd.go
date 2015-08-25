package herd

import (
	"runtime"
)

func newHerd(imageServerAddress string) *Herd {
	var herd Herd
	herd.imageServerAddress = imageServerAddress
	herd.makeConnectionSemaphore = make(chan bool, 1000)
	herd.pollSemaphore = make(chan bool, runtime.NumCPU()*2)
	return &herd
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
		herd.waitForCompletion()
		return true
	}
	sub := herd.subsByIndex[herd.nextSubToPoll]
	herd.nextSubToPoll++
	herd.makeConnectionSemaphore <- true
	go func() {
		sub.connect()
		if sub.connection != nil {
			herd.pollSemaphore <- true
			sub.poll()
			<-herd.pollSemaphore
		}
		<-herd.makeConnectionSemaphore
	}()
	return false
}
