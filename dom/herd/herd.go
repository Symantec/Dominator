package herd

import (
	"errors"
	"flag"
	"github.com/Symantec/Dominator/dom/images"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/cpusharer"
	filegenclient "github.com/Symantec/Dominator/lib/filegen/client"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/url"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/tricorder/go/tricorder"
	"log"
	"runtime"
	"time"
)

var (
	pollSlotsPerCPU = flag.Uint("pollSlotsPerCPU", 100,
		"Number of poll slots per CPU")
)

func newHerd(imageServerAddress string, objectServer objectserver.ObjectServer,
	metricsDir *tricorder.DirectorySpec, logger *log.Logger) *Herd {
	var herd Herd
	herd.imageManager = images.New(imageServerAddress, logger)
	herd.objectServer = objectServer
	herd.computedFilesManager = filegenclient.New(objectServer, logger)
	herd.logger = logger
	herd.configurationForSubs.ScanExclusionList =
		constants.ScanExcludeList
	herd.subsByName = make(map[string]*Sub)
	numPollSlots := uint(runtime.NumCPU()) * *pollSlotsPerCPU
	herd.pollSemaphore = make(chan struct{}, numPollSlots)
	herd.pushSemaphore = make(chan struct{}, runtime.NumCPU())
	herd.cpuSharer = cpusharer.NewFifoCpuSharer()
	herd.currentScanStartTime = time.Now()
	herd.setupMetrics(metricsDir)
	return &herd
}

func (herd *Herd) clearSafetyShutoff(hostname string) error {
	herd.Lock()
	sub, ok := herd.subsByName[hostname]
	herd.Unlock()
	if !ok {
		return errors.New("unknown sub: " + hostname)
	}
	return sub.clearSafetyShutoff()
}

func (herd *Herd) configureSubs(configuration subproto.Configuration) error {
	herd.Lock()
	defer herd.Unlock()
	herd.configurationForSubs = configuration
	return nil
}

func (herd *Herd) disableUpdates(username, reason string) error {
	if reason == "" {
		return errors.New("error disabling updates: no reason given")
	}
	herd.updatesDisabledBy = username
	herd.updatesDisabledReason = reason
	herd.updatesDisabledTime = time.Now()
	return nil
}

func (herd *Herd) enableUpdates() error {
	herd.updatesDisabledReason = ""
	return nil
}

func (herd *Herd) getSubsConfiguration() subproto.Configuration {
	herd.RLockWithTimeout(time.Minute)
	defer herd.RUnlock()
	return herd.configurationForSubs
}

func (herd *Herd) lockWithTimeout(timeout time.Duration) {
	timeoutFunction(herd.Lock, timeout)
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
	herd.cpuSharer.Go(func() {
		if !sub.tryMakeBusy() {
			return
		}
		sub.connectAndPoll()
		sub.makeUnbusy()
	})
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

func (herd *Herd) getSub(name string) *Sub {
	herd.RLock()
	defer herd.RUnlock()
	return herd.subsByName[name]
}

func (herd *Herd) getReachableSelector(parsedQuery url.ParsedQuery) (
	func(*Sub) bool, error) {
	duration, err := parsedQuery.Last()
	if err != nil {
		return nil, err
	}
	return rDuration(duration).selector, nil
}

func (herd *Herd) rLockWithTimeout(timeout time.Duration) {
	timeoutFunction(herd.RLock, timeout)
}

func (herd *Herd) setDefaultImage(imageName string) error {
	if imageName == "" {
		herd.Lock()
		defer herd.Unlock()
		herd.defaultImageName = ""
		// Cancel blocking operations by affected subs.
		for _, sub := range herd.subsByIndex {
			if sub.mdb.RequiredImage != "" {
				sub.sendCancel()
				sub.status = statusImageUndefined
			}
		}
		return nil
	}
	if imageName == herd.defaultImageName {
		return nil
	}
	herd.Lock()
	herd.nextDefaultImageName = imageName
	herd.Unlock()
	doLockedCleanup := true
	defer func() {
		if doLockedCleanup {
			herd.Lock()
			herd.nextDefaultImageName = ""
			herd.Unlock()
		}
	}()
	img, err := herd.imageManager.Get(imageName, true)
	if err != nil {
		return err
	}
	if img == nil {
		return errors.New("unknown image: " + imageName)
	}
	if img.Filter != nil {
		return errors.New("only sparse images can be set as default")
	}
	if len(img.FileSystem.InodeTable) > 100 {
		return errors.New("cannot set default image with more than 100 inodes")
	}
	doLockedCleanup = false
	herd.Lock()
	defer herd.Unlock()
	herd.defaultImageName = imageName
	herd.nextDefaultImageName = ""
	for _, sub := range herd.subsByIndex {
		if sub.mdb.RequiredImage == "" {
			sub.sendCancel()
			if sub.status == statusSynced { // Synced to previous default image.
				sub.status = statusWaitingToPoll
			}
			if sub.status == statusImageUndefined {
				sub.status = statusWaitingToPoll
			}
		}
	}
	return nil
}

func timeoutFunction(f func(), timeout time.Duration) {
	if timeout < 0 {
		f()
		return
	}
	completionChannel := make(chan struct{})
	go func() {
		f()
		completionChannel <- struct{}{}
	}()
	timer := time.NewTimer(timeout)
	select {
	case <-completionChannel:
		if !timer.Stop() {
			<-timer.C
		}
		return
	case <-timer.C:
		panic("timeout")
	}
}
