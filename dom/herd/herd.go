package herd

import (
	"errors"
	"flag"
	"github.com/Symantec/Dominator/dom/images"
	"github.com/Symantec/Dominator/lib/constants"
	filegenclient "github.com/Symantec/Dominator/lib/filegen/client"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/url"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"log"
	"runtime"
	"syscall"
	"time"
)

var (
	pollSlotsPerCPU = flag.Uint("pollSlotsPerCPU", 100,
		"Number of poll slots per CPU")
)

func newHerd(imageServerAddress string, objectServer objectserver.ObjectServer,
	logger *log.Logger) *Herd {
	var herd Herd
	herd.imageManager = images.New(imageServerAddress, logger)
	herd.objectServer = objectServer
	herd.computedFilesManager = filegenclient.New(objectServer, logger)
	herd.logger = logger
	herd.configurationForSubs.ScanSpeedPercent =
		constants.DefaultScanSpeedPercent
	herd.configurationForSubs.NetworkSpeedPercent =
		constants.DefaultNetworkSpeedPercent
	herd.configurationForSubs.ScanExclusionList =
		constants.ScanExcludeList
	herd.subsByName = make(map[string]*Sub)
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
	herd.connectionSemaphore = make(chan struct{}, maxConnAttempts)
	numPollSlots := uint(runtime.NumCPU()) * *pollSlotsPerCPU
	herd.pollSemaphore = make(chan struct{}, numPollSlots)
	herd.pushSemaphore = make(chan struct{}, runtime.NumCPU())
	herd.computeSemaphore = make(chan struct{}, runtime.NumCPU())
	herd.currentScanStartTime = time.Now()
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
	herd.RLock()
	defer herd.RUnlock()
	return herd.configurationForSubs
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
	herd.connectionSemaphore <- struct{}{}
	go func() {
		defer func() { <-herd.connectionSemaphore }()
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

func (herd *Herd) getReachableSelector(parsedQuery url.ParsedQuery) (
	func(*Sub) bool, error) {
	duration, err := parsedQuery.Last()
	if err != nil {
		return nil, err
	}
	return rDuration(duration).selector, nil
}

func (herd *Herd) setDefaultImage(imageName string) error {
	if imageName == "" {
		herd.defaultImageName = ""
		return nil
	}
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
	herd.defaultImageName = imageName
	return nil
}
