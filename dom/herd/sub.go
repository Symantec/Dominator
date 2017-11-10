package herd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/Symantec/Dominator/dom/lib"
	"github.com/Symantec/Dominator/lib/constants"
	filegenclient "github.com/Symantec/Dominator/lib/filegen/client"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/resourcepool"
	"github.com/Symantec/Dominator/lib/srpc"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
)

var (
	logUnknownSubConnectErrors = flag.Bool("logUnknownSubConnectErrors", false,
		"If true, log unknown sub connection errors")
	showIP = flag.Bool("showIP", false,
		"If true, prefer to show IP address from MDB if available")
	useIP = flag.Bool("useIP", true,
		"If true, prefer to use IP address from MDB if available")

	subPortNumber = fmt.Sprintf(":%d", constants.SubPortNumber)
	zeroHash      hash.Hash
)

func (sub *Sub) string() string {
	if *showIP && sub.mdb.IpAddress != "" {
		return sub.mdb.IpAddress
	}
	return sub.mdb.Hostname
}

func (sub *Sub) address() string {
	if *useIP && sub.mdb.IpAddress != "" {
		hostInstance := strings.SplitN(sub.mdb.Hostname, "*", 2)
		if len(hostInstance) > 1 {
			return sub.mdb.IpAddress + "*" + hostInstance[1] + subPortNumber
		}
		return sub.mdb.IpAddress + subPortNumber
	}
	return sub.mdb.Hostname + subPortNumber
}

func (sub *Sub) getComputedFiles(im *image.Image) []filegenclient.ComputedFile {
	if im == nil {
		return nil
	}
	numComputed := im.FileSystem.NumComputedRegularInodes()
	if numComputed < 1 {
		return nil
	}
	computedFiles := make([]filegenclient.ComputedFile, 0, numComputed)
	inodeToFilenamesTable := im.FileSystem.InodeToFilenamesTable()
	for inum, inode := range im.FileSystem.InodeTable {
		if inode, ok := inode.(*filesystem.ComputedRegularInode); ok {
			if filenames, ok := inodeToFilenamesTable[inum]; ok {
				if len(filenames) == 1 {
					computedFiles = append(computedFiles,
						filegenclient.ComputedFile{filenames[0], inode.Source})
				}
			}
		}
	}
	return computedFiles
}

func (sub *Sub) tryMakeBusy() bool {
	sub.busyFlagMutex.Lock()
	defer sub.busyFlagMutex.Unlock()
	if sub.busy {
		return false
	}
	sub.busyStartTime = time.Now()
	sub.busy = true
	return true
}

func (sub *Sub) makeUnbusy() {
	sub.busyFlagMutex.Lock()
	defer sub.busyFlagMutex.Unlock()
	sub.busyStopTime = time.Now()
	sub.busy = false
}

func (sub *Sub) connectAndPoll() {
	sub.loadConfiguration()
	if sub.processFileUpdates() {
		sub.generationCount = 0 // Force a full poll.
	}
	sub.deletingFlagMutex.Lock()
	if sub.deleting {
		sub.deletingFlagMutex.Unlock()
		return
	}
	if sub.clientResource == nil {
		sub.clientResource = srpc.NewClientResource("tcp", sub.address())
	}
	sub.deletingFlagMutex.Unlock()
	previousStatus := sub.status
	timer := time.AfterFunc(time.Second, func() {
		sub.publishedStatus = sub.status
	})
	defer func() {
		timer.Stop()
		sub.publishedStatus = sub.status
	}()
	sub.lastConnectionStartTime = time.Now()
	srpcClient, err := sub.clientResource.GetHTTPWithDialer(sub.cancelChannel,
		sub.herd.dialer)
	dialReturnedTime := time.Now()
	if err != nil {
		sub.isInsecure = false
		sub.pollTime = time.Time{}
		if err == resourcepool.ErrorResourceLimitExceeded {
			return
		}
		if err, ok := err.(*net.OpError); ok {
			if _, ok := err.Err.(*net.DNSError); ok {
				sub.status = statusDNSError
				return
			}
			if err.Timeout() {
				sub.status = statusConnectTimeout
				return
			}
		}
		if err == srpc.ErrorConnectionRefused {
			sub.status = statusConnectionRefused
			return
		}
		if err == srpc.ErrorNoRouteToHost {
			sub.status = statusNoRouteToHost
			return
		}
		if err == srpc.ErrorMissingCertificate {
			sub.lastReachableTime = dialReturnedTime
			sub.status = statusMissingCertificate
			return
		}
		if err == srpc.ErrorBadCertificate {
			sub.lastReachableTime = dialReturnedTime
			sub.status = statusBadCertificate
			return
		}
		sub.status = statusFailedToConnect
		if *logUnknownSubConnectErrors {
			sub.herd.logger.Println(err)
		}
		return
	}
	defer srpcClient.Put()
	if srpcClient.IsEncrypted() {
		sub.isInsecure = false
	} else {
		sub.isInsecure = true
	}
	sub.lastReachableTime = dialReturnedTime
	sub.lastConnectionSucceededTime = dialReturnedTime
	sub.lastConnectDuration =
		sub.lastConnectionSucceededTime.Sub(sub.lastConnectionStartTime)
	connectDistribution.Add(sub.lastConnectDuration)
	waitStartTime := time.Now()
	sub.herd.cpuSharer.ReleaseCpu()
	select {
	case sub.herd.pollSemaphore <- struct{}{}:
		sub.herd.cpuSharer.GrabCpu()
		break
	case <-sub.cancelChannel:
		sub.herd.cpuSharer.GrabCpu()
		return
	}
	pollWaitTimeDistribution.Add(time.Since(waitStartTime))
	sub.status = statusPolling
	sub.poll(srpcClient, previousStatus)
	<-sub.herd.pollSemaphore
}

func (sub *Sub) loadConfiguration() {
	// Get a stable copy of the configuration.
	newRequiredImageName := sub.mdb.RequiredImage
	if newRequiredImageName == "" {
		newRequiredImageName = sub.herd.defaultImageName
	}
	if newRequiredImageName != sub.requiredImageName {
		sub.computedInodes = nil
	}
	sub.herd.cpuSharer.ReleaseCpu()
	defer sub.herd.cpuSharer.GrabCpu()
	sub.requiredImageName = newRequiredImageName
	sub.requiredImage = sub.herd.imageManager.GetNoError(sub.requiredImageName)
	sub.plannedImageName = sub.mdb.PlannedImage
	sub.plannedImage = sub.herd.imageManager.GetNoError(sub.plannedImageName)
}

func (sub *Sub) processFileUpdates() bool {
	haveUpdates := false
	for {
		image := sub.requiredImage
		if image != nil && sub.computedInodes == nil {
			sub.computedInodes = make(map[string]*filesystem.RegularInode)
			sub.deletingFlagMutex.Lock()
			if sub.deleting {
				sub.deletingFlagMutex.Unlock()
				return false
			}
			computedFiles := sub.getComputedFiles(image)
			sub.herd.cpuSharer.ReleaseCpu()
			sub.herd.computedFilesManager.Update(
				filegenclient.Machine{sub.mdb, computedFiles})
			sub.herd.cpuSharer.GrabCpu()
			sub.deletingFlagMutex.Unlock()
		}
		select {
		case fileInfos := <-sub.fileUpdateChannel:
			if image == nil {
				continue
			}
			filenameToInodeTable := image.FileSystem.FilenameToInodeTable()
			for _, fileInfo := range fileInfos {
				if fileInfo.Hash == zeroHash {
					continue // No object.
				}
				inum, ok := filenameToInodeTable[fileInfo.Pathname]
				if !ok {
					continue
				}
				genericInode, ok := image.FileSystem.InodeTable[inum]
				if !ok {
					continue
				}
				cInode, ok := genericInode.(*filesystem.ComputedRegularInode)
				if !ok {
					continue
				}
				rInode := &filesystem.RegularInode{
					Mode:         cInode.Mode,
					Uid:          cInode.Uid,
					Gid:          cInode.Gid,
					MtimeSeconds: -1, // The time is set during the compute.
					Size:         fileInfo.Length,
					Hash:         fileInfo.Hash,
				}
				sub.computedInodes[fileInfo.Pathname] = rInode
				haveUpdates = true
			}
		default:
			return haveUpdates
		}
	}
	return haveUpdates
}

func (sub *Sub) poll(srpcClient *srpc.Client, previousStatus subStatus) {
	// If the planned image has just become available, force a full poll.
	if previousStatus == statusSynced &&
		!sub.havePlannedImage &&
		sub.plannedImage != nil {
		sub.havePlannedImage = true
		sub.generationCount = 0 // Force a full poll.
	}
	// If the computed files have changed since the last sync, force a full poll
	if previousStatus == statusSynced &&
		sub.computedFilesChangeTime.After(sub.lastSyncTime) {
		sub.generationCount = 0 // Force a full poll.
	}
	// If the last update was disabled and updates are enabled now, force a full
	// poll.
	if previousStatus == statusUpdatesDisabled &&
		sub.herd.updatesDisabledReason == "" && !sub.mdb.DisableUpdates {
		sub.generationCount = 0 // Force a full poll.
	}
	// If the last update was disabled due to a safety check and there is a
	// pending SafetyClear, force a full poll to re-compute the update.
	if previousStatus == statusUnsafeUpdate && sub.pendingSafetyClear {
		sub.generationCount = 0 // Force a full poll.
	}
	var request subproto.PollRequest
	request.HaveGeneration = sub.generationCount
	var reply subproto.PollResponse
	haveImage := false
	if sub.requiredImage == nil {
		request.ShortPollOnly = true
	} else {
		haveImage = true
	}
	logger := sub.herd.logger
	sub.lastPollStartTime = time.Now()
	if err := client.CallPoll(srpcClient, request, &reply); err != nil {
		srpcClient.Close()
		if err == io.EOF {
			return
		}
		sub.pollTime = time.Time{}
		if err == srpc.ErrorAccessToMethodDenied {
			sub.status = statusPollDenied
		} else {
			sub.status = statusFailedToPoll
		}
		logger.Printf("Error calling %s.Poll(): %s\n", sub, err)
		return
	}
	sub.lastPollSucceededTime = time.Now()
	sub.lastSuccessfulImageName = reply.LastSuccessfulImageName
	if reply.GenerationCount == 0 {
		sub.reclaim()
		sub.generationCount = 0
	}
	sub.lastScanDuration = reply.DurationOfLastScan
	if fs := reply.FileSystem; fs == nil {
		sub.lastPollWasFull = false
		sub.lastShortPollDuration =
			sub.lastPollSucceededTime.Sub(sub.lastPollStartTime)
		shortPollDistribution.Add(sub.lastShortPollDuration)
		if !sub.startTime.Equal(reply.StartTime) {
			sub.generationCount = 0 // Sub has restarted: force a full poll.
		}
		if sub.freeSpaceThreshold != nil && reply.FreeSpace != nil {
			if *reply.FreeSpace > *sub.freeSpaceThreshold {
				sub.generationCount = 0 // Force a full poll for next time.
			}
		}
	} else {
		sub.lastPollWasFull = true
		sub.freeSpaceThreshold = nil
		if err := fs.RebuildInodePointers(); err != nil {
			sub.status = statusFailedToPoll
			logger.Printf("Error building pointers for: %s %s\n", sub, err)
			return
		}
		fs.BuildEntryMap()
		sub.fileSystem = fs
		sub.objectCache = reply.ObjectCache
		sub.generationCount = reply.GenerationCount
		sub.lastFullPollDuration =
			sub.lastPollSucceededTime.Sub(sub.lastPollStartTime)
		fullPollDistribution.Add(sub.lastFullPollDuration)
	}
	sub.startTime = reply.StartTime
	sub.pollTime = reply.PollTime
	sub.updateConfiguration(srpcClient, reply)
	if reply.FetchInProgress {
		sub.status = statusFetching
		return
	}
	if reply.UpdateInProgress {
		sub.status = statusUpdating
		return
	}
	if reply.GenerationCount < 1 {
		sub.status = statusSubNotReady
		return
	}
	if previousStatus == statusFetching && reply.LastFetchError != "" {
		logger.Printf("Fetch failure for: %s: %s\n", sub, reply.LastFetchError)
		sub.status = statusFailedToFetch
		if sub.fileSystem == nil {
			sub.generationCount = 0 // Force a full poll next cycle.
			return
		}
	}
	if previousStatus == statusUpdating {
		// Transition from updating to update ended (may be partial/failed).
		if reply.LastUpdateError != "" {
			logger.Printf("Update failure for: %s: %s\n",
				sub, reply.LastUpdateError)
			sub.status = statusFailedToUpdate
		} else {
			sub.status = statusWaitingForNextFullPoll
		}
		sub.scanCountAtLastUpdateEnd = reply.ScanCount
		sub.reclaim()
		return
	}
	if sub.checkCancel() {
		// Configuration change pending: skip further processing. Do not reclaim
		// file-system and objectcache data: it will speed up the next Poll.
		return
	}
	if !haveImage {
		if sub.requiredImageName == "" {
			sub.status = statusImageUndefined
		} else {
			sub.status = statusImageNotReady
		}
		return
	}
	if previousStatus == statusFailedToUpdate ||
		previousStatus == statusWaitingForNextFullPoll {
		if sub.scanCountAtLastUpdateEnd == reply.ScanCount {
			// Need to wait until sub has performed a new scan.
			if sub.fileSystem != nil {
				sub.reclaim()
			}
			sub.status = previousStatus
			return
		}
		if sub.fileSystem == nil {
			// Force a full poll next cycle so that we can see the state of the
			// sub.
			sub.generationCount = 0
			sub.status = previousStatus
			return
		}
	}
	if sub.fileSystem == nil {
		sub.status = previousStatus
		return
	}
	if idle, status := sub.fetchMissingObjects(srpcClient, sub.requiredImage,
		reply.FreeSpace, true); !idle {
		sub.status = status
		sub.reclaim()
		return
	}
	sub.status = statusComputingUpdate
	if idle, status := sub.sendUpdate(srpcClient); !idle {
		sub.status = status
		sub.reclaim()
		return
	}
	if idle, status := sub.fetchMissingObjects(srpcClient, sub.plannedImage,
		reply.FreeSpace, false); !idle {
		if status != statusImageNotReady && status != statusNotEnoughFreeSpace {
			sub.status = status
			sub.reclaim()
			return
		}
	}
	if previousStatus == statusWaitingForNextFullPoll &&
		!sub.lastUpdateTime.IsZero() {
		sub.lastSyncTime = time.Now()
	}
	sub.status = statusSynced
	sub.cleanup(srpcClient)
	sub.reclaim()
}

func (sub *Sub) reclaim() {
	sub.fileSystem = nil  // Mark memory for reclaim.
	sub.objectCache = nil // Mark memory for reclaim.
}

func (sub *Sub) updateConfiguration(srpcClient *srpc.Client,
	pollReply subproto.PollResponse) {
	if pollReply.ScanCount < 1 {
		return
	}
	sub.herd.RLockWithTimeout(time.Minute)
	newConf := sub.herd.configurationForSubs
	sub.herd.RUnlock()
	if newConf.CpuPercent < 1 {
		newConf.CpuPercent = pollReply.CurrentConfiguration.CpuPercent
	}
	if newConf.NetworkSpeedPercent < 1 {
		newConf.NetworkSpeedPercent =
			pollReply.CurrentConfiguration.NetworkSpeedPercent
	}
	if newConf.ScanSpeedPercent < 1 {
		newConf.ScanSpeedPercent =
			pollReply.CurrentConfiguration.ScanSpeedPercent
	}
	if compareConfigs(pollReply.CurrentConfiguration, newConf) {
		return
	}
	if err := client.SetConfiguration(srpcClient, newConf); err != nil {
		srpcClient.Close()
		logger := sub.herd.logger
		logger.Printf("Error setting configuration for sub: %s: %s\n",
			sub, err)
		return
	}
}

func compareConfigs(oldConf, newConf subproto.Configuration) bool {
	if newConf.CpuPercent != oldConf.CpuPercent {
		return false
	}
	if newConf.NetworkSpeedPercent != oldConf.NetworkSpeedPercent {
		return false
	}
	if newConf.ScanSpeedPercent != oldConf.ScanSpeedPercent {
		return false
	}
	if len(newConf.ScanExclusionList) != len(oldConf.ScanExclusionList) {
		return false
	}
	for index, newString := range newConf.ScanExclusionList {
		if newString != oldConf.ScanExclusionList[index] {
			return false
		}
	}
	return true
}

// Returns true if all required objects are available.
func (sub *Sub) fetchMissingObjects(srpcClient *srpc.Client, image *image.Image,
	freeSpace *uint64, pushComputedFiles bool) (
	bool, subStatus) {
	if image == nil {
		return false, statusImageNotReady
	}
	logger := sub.herd.logger
	subObj := lib.Sub{
		Hostname:       sub.mdb.Hostname,
		Client:         srpcClient,
		FileSystem:     sub.fileSystem,
		ComputedInodes: sub.computedInodes,
		ObjectCache:    sub.objectCache,
		ObjectGetter:   sub.herd.objectServer}
	objectsToFetch, objectsToPush := lib.BuildMissingLists(subObj, image,
		pushComputedFiles, false, logger)
	if objectsToPush == nil {
		return false, statusMissingComputedFile
	}
	var returnAvailable bool = true
	var returnStatus subStatus = statusSynced
	if len(objectsToFetch) > 0 {
		if !sub.checkForEnoughSpace(freeSpace, objectsToFetch) {
			return false, statusNotEnoughFreeSpace
		}
		logger.Printf("Calling %s:Subd.Fetch() for: %d objects\n",
			sub, len(objectsToFetch))
		err := client.Fetch(srpcClient, sub.herd.imageManager.String(),
			objectcache.ObjectMapToCache(objectsToFetch))
		if err != nil {
			srpcClient.Close()
			logger.Printf("Error calling %s:Subd.Fetch(): %s\n", sub, err)
			if err == srpc.ErrorAccessToMethodDenied {
				return false, statusFetchDenied
			}
			return false, statusFailedToFetch
		}
		returnAvailable = false
		returnStatus = statusFetching
	}
	if len(objectsToPush) > 0 {
		sub.herd.cpuSharer.GrabSemaphore(sub.herd.pushSemaphore)
		defer func() { <-sub.herd.pushSemaphore }()
		sub.status = statusPushing
		err := lib.PushObjects(subObj, objectsToPush, logger)
		if err != nil {
			if err == srpc.ErrorAccessToMethodDenied {
				return false, statusPushDenied
			}
			if err == lib.ErrorFailedToGetObject {
				return false, statusFailedToGetObject
			}
			return false, statusFailedToPush
		}
		if returnAvailable {
			// Update local copy of objectcache, since there will not be
			// another Poll() before the update computation.
			for hashVal := range objectsToPush {
				sub.objectCache = append(sub.objectCache, hashVal)
			}
		}
	}
	return returnAvailable, returnStatus
}

// Returns true if no update needs to be performed.
func (sub *Sub) sendUpdate(srpcClient *srpc.Client) (bool, subStatus) {
	logger := sub.herd.logger
	if !sub.pendingSafetyClear {
		// Perform a cheap safety check: if over half the inodes will be deleted
		// then mark the update as unsafe.
		if sub.requiredImage.Filter != nil &&
			len(sub.fileSystem.InodeTable)>>1 >
				len(sub.requiredImage.FileSystem.InodeTable) {
			return false, statusUnsafeUpdate
		}
	}
	var request subproto.UpdateRequest
	var reply subproto.UpdateResponse
	if idle, missing := sub.buildUpdateRequest(&request); missing {
		return false, statusMissingComputedFile
	} else if idle {
		return true, statusSynced
	}
	if sub.mdb.DisableUpdates || sub.herd.updatesDisabledReason != "" {
		return false, statusUpdatesDisabled
	}
	sub.status = statusSendingUpdate
	sub.lastUpdateTime = time.Now()
	logger.Printf("Calling %s:Subd.Update() for image: %s\n",
		sub, sub.requiredImageName)
	if err := client.CallUpdate(srpcClient, request, &reply); err != nil {
		srpcClient.Close()
		logger.Printf("Error calling %s:Subd.Update(): %s\n", sub, err)
		if err == srpc.ErrorAccessToMethodDenied {
			return false, statusUpdateDenied
		}
		return false, statusFailedToUpdate
	}
	sub.pendingSafetyClear = false
	return false, statusUpdating
}

func (sub *Sub) cleanup(srpcClient *srpc.Client) {
	logger := sub.herd.logger
	unusedObjects := make(map[hash.Hash]bool)
	for _, hash := range sub.objectCache {
		unusedObjects[hash] = false // Potential cleanup candidate.
	}
	for _, inode := range sub.fileSystem.InodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			if inode.Size > 0 {
				if _, ok := unusedObjects[inode.Hash]; ok {
					unusedObjects[inode.Hash] = true // Must clean this one up.
				}
			}
		}
	}
	image := sub.plannedImage
	if image != nil {
		for _, inode := range image.FileSystem.InodeTable {
			if inode, ok := inode.(*filesystem.RegularInode); ok {
				if inode.Size > 0 {
					if clean, ok := unusedObjects[inode.Hash]; !clean && ok {
						delete(unusedObjects, inode.Hash)
					}
				}
			}
		}
	}
	if len(unusedObjects) < 1 {
		return
	}
	hashes := make([]hash.Hash, 0, len(unusedObjects))
	for hash := range unusedObjects {
		hashes = append(hashes, hash)
	}
	if err := client.Cleanup(srpcClient, hashes); err != nil {
		srpcClient.Close()
		logger.Printf("Error calling %s:Subd.Cleanup(): %s\n", sub, err)
	}
}

func (sub *Sub) checkForEnoughSpace(freeSpace *uint64,
	objects map[hash.Hash]uint64) bool {
	if freeSpace == nil {
		sub.freeSpaceThreshold = nil
		return true // Don't know, assume OK.
	}
	var totalUsage uint64
	for _, size := range objects {
		usage := (size >> 12) << 12
		if usage < size {
			usage += 1 << 12
		}
		totalUsage += usage
	}
	if *freeSpace > totalUsage {
		sub.freeSpaceThreshold = nil
		return true
	}
	sub.freeSpaceThreshold = &totalUsage
	return false
}

func (sub *Sub) clearSafetyShutoff() error {
	if sub.status != statusUnsafeUpdate {
		return errors.New("no pending unsafe update")
	}
	sub.pendingSafetyClear = true
	return nil
}

func (sub *Sub) checkCancel() bool {
	select {
	case <-sub.cancelChannel:
		return true
	default:
		return false
	}
}

func (sub *Sub) sendCancel() {
	select {
	case sub.cancelChannel <- struct{}{}:
	default:
	}
}
