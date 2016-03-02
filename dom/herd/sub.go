package herd

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"net"
	"runtime"
	"strings"
	"time"
)

var (
	subConnectTimeout = flag.Uint("subConnectTimeout", 15,
		"Timeout in seconds for sub connections. If zero, OS timeout is used")
	logUnknownSubConnectErrors = flag.Bool("logUnknownSubConnectErrors", false,
		"If true, log unknown sub connection errors")
)

func (sub *Sub) tryMakeBusy() bool {
	sub.busyMutex.Lock()
	defer sub.busyMutex.Unlock()
	if sub.busy {
		return false
	}
	sub.busyStartTime = time.Now()
	sub.busy = true
	return true
}

func (sub *Sub) makeUnbusy() {
	sub.busyMutex.Lock()
	defer sub.busyMutex.Unlock()
	sub.busyStopTime = time.Now()
	sub.busy = false
}

func (sub *Sub) connectAndPoll() {
	previousStatus := sub.status
	sub.status = statusConnecting
	hostname := strings.SplitN(sub.hostname, "*", 2)[0]
	address := fmt.Sprintf("%s:%d", hostname, constants.SubPortNumber)
	sub.lastConnectionStartTime = time.Now()
	srpcClient, err := srpc.DialHTTP("tcp", address,
		time.Second*time.Duration(*subConnectTimeout))
	dialReturnedTime := time.Now()
	if err != nil {
		sub.pollTime = time.Time{}
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
	defer srpcClient.Close()
	sub.status = statusWaitingToPoll
	sub.lastReachableTime = dialReturnedTime
	sub.lastConnectionSucceededTime = dialReturnedTime
	sub.lastConnectDuration =
		sub.lastConnectionSucceededTime.Sub(sub.lastConnectionStartTime)
	connectDistribution.Add(sub.lastConnectDuration)
	sub.herd.pollSemaphore <- struct{}{}
	sub.status = statusPolling
	sub.poll(srpcClient, previousStatus)
	<-sub.herd.pollSemaphore
}

func (sub *Sub) poll(srpcClient *srpc.Client, previousStatus subStatus) {
	// If the planned image has just become available, force a full poll.
	if previousStatus == statusSynced &&
		!sub.havePlannedImage &&
		sub.herd.getImageNoError(sub.plannedImage) != nil {
		sub.havePlannedImage = true
		sub.generationCount = 0 // Force a full poll.
	}
	// If the computed files have changed since the last sync, force a full poll
	if previousStatus == statusSynced &&
		sub.computedFilesChangeTime.After(sub.lastSyncTime) {
		sub.generationCount = 0 // Force a full poll.
	}
	var request subproto.PollRequest
	request.HaveGeneration = sub.generationCount
	var reply subproto.PollResponse
	sub.lastPollStartTime = time.Now()
	haveImage := false
	if sub.herd.getImageNoError(sub.requiredImage) == nil {
		request.ShortPollOnly = true
	} else {
		haveImage = true
	}
	logger := sub.herd.logger
	if err := client.CallPoll(srpcClient, request, &reply); err != nil {
		sub.pollTime = time.Time{}
		if err == srpc.ErrorAccessToMethodDenied {
			sub.status = statusPollDenied
		} else {
			sub.status = statusFailedToPoll
		}
		logger.Printf("Error calling %s.Poll()\t%s\n", sub.hostname, err)
		return
	}
	sub.lastPollSucceededTime = time.Now()
	if reply.GenerationCount == 0 {
		sub.reclaim()
		sub.generationCount = 0
	}
	if fs := reply.FileSystem; fs == nil {
		sub.lastShortPollDuration =
			sub.lastPollSucceededTime.Sub(sub.lastPollStartTime)
		shortPollDistribution.Add(sub.lastShortPollDuration)
		if !sub.startTime.Equal(reply.StartTime) {
			sub.generationCount = 0 // Sub has restarted: force a full poll.
		}
	} else {
		if err := fs.RebuildInodePointers(); err != nil {
			sub.status = statusFailedToPoll
			logger.Printf("Error building pointers for: %s %s\n",
				sub.hostname, err)
			return
		}
		fs.BuildEntryMap()
		sub.fileSystem = fs
		sub.objectCache = reply.ObjectCache
		sub.generationCount = reply.GenerationCount
		sub.lastFullPollDuration =
			sub.lastPollSucceededTime.Sub(sub.lastPollStartTime)
		fullPollDistribution.Add(sub.lastFullPollDuration)
		logger.Printf("Polled: %s, GenerationCount=%d\n",
			sub.hostname, reply.GenerationCount)
	}
	sub.startTime = reply.StartTime
	sub.pollTime = reply.PollTime
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
		logger.Printf("Fetch failure for: %s: %s\n",
			sub.hostname, reply.LastFetchError)
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
				sub.hostname, reply.LastUpdateError)
			sub.status = statusFailedToUpdate
		} else {
			sub.status = statusWaitingForNextFullPoll
		}
		sub.scanCountAtLastUpdateEnd = reply.ScanCount
		sub.reclaim()
		return
	}
	if !haveImage {
		sub.status = statusImageNotReady
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
	if idle, status := sub.fetchMissingObjects(srpcClient,
		sub.requiredImage); !idle {
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
	if idle, status := sub.fetchMissingObjects(srpcClient,
		sub.plannedImage); !idle {
		if status != statusImageNotReady {
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
	sub.cleanup(srpcClient, sub.plannedImage)
	sub.reclaim()
}

func (sub *Sub) reclaim() {
	sub.fileSystem = nil  // Mark memory for reclaim.
	sub.objectCache = nil // Mark memory for reclaim.
	runtime.GC()          // Reclaim now.
}

// Returns true if all required objects are available.
func (sub *Sub) fetchMissingObjects(srpcClient *srpc.Client, imageName string) (
	bool, subStatus) {
	image := sub.herd.getImageNoError(imageName)
	if image == nil {
		return false, statusImageNotReady
	}
	missingObjects := make(map[hash.Hash]struct{})
	for _, inode := range image.FileSystem.InodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			if inode.Size > 0 {
				missingObjects[inode.Hash] = struct{}{}
			}
		}
	}
	for _, hash := range sub.objectCache {
		delete(missingObjects, hash)
	}
	for _, inode := range sub.fileSystem.InodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			if inode.Size > 0 {
				delete(missingObjects, inode.Hash)
			}
		}
	}
	if len(missingObjects) < 1 {
		return true, statusSynced
	}
	logger := sub.herd.logger
	logger.Printf("Calling %s.Fetch() for: %d objects\n",
		sub.hostname, len(missingObjects))
	var request subproto.FetchRequest
	var reply subproto.FetchResponse
	request.ServerAddress = sub.herd.imageServerAddress
	for hash := range missingObjects {
		request.Hashes = append(request.Hashes, hash)
	}
	if err := client.CallFetch(srpcClient, request, &reply); err != nil {
		logger.Printf("Error calling %s.Fetch()\t%s\n", sub.hostname, err)
		if err == srpc.ErrorAccessToMethodDenied {
			return false, statusFetchDenied
		}
		return false, statusFailedToFetch
	}
	return false, statusFetching
}

// Returns true if no update needs to be performed.
func (sub *Sub) sendUpdate(srpcClient *srpc.Client) (bool, subStatus) {
	logger := sub.herd.logger
	var request subproto.UpdateRequest
	var reply subproto.UpdateResponse
	if idle, missing := sub.buildUpdateRequest(&request); missing {
		return false, statusMissingComputedFile
	} else if idle {
		return true, statusSynced
	}
	sub.status = statusSendingUpdate
	sub.lastUpdateTime = time.Now()
	if err := client.CallUpdate(srpcClient, request, &reply); err != nil {
		logger.Printf("Error calling %s:Subd.Update()\t%s\n", sub.hostname, err)
		if err == srpc.ErrorAccessToMethodDenied {
			return false, statusUpdateDenied
		}
		return false, statusFailedToUpdate
	}
	return false, statusUpdating
}

func (sub *Sub) cleanup(srpcClient *srpc.Client, plannedImageName string) {
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
	image := sub.herd.getImageNoError(plannedImageName)
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
	var request subproto.CleanupRequest
	var reply subproto.CleanupResponse
	request.Hashes = make([]hash.Hash, 0, len(unusedObjects))
	for hash := range unusedObjects {
		request.Hashes = append(request.Hashes, hash)
	}
	if err := client.CallCleanup(srpcClient, request, &reply); err != nil {
		logger.Printf("Error calling %s:Subd.Cleanup()\t%s\n",
			sub.hostname, err)
	}
}
