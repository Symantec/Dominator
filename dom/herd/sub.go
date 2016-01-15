package herd

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"runtime"
	"strings"
	"time"
)

func (sub *Sub) tryMakeBusy() bool {
	sub.busyMutex.Lock()
	defer sub.busyMutex.Unlock()
	if sub.busy {
		return false
	}
	sub.busy = true
	return true
}

func (sub *Sub) makeUnbusy() {
	sub.busyMutex.Lock()
	defer sub.busyMutex.Unlock()
	sub.busy = false
}

func (sub *Sub) connectAndPoll() {
	previousStatus := sub.status
	sub.status = statusConnecting
	hostname := strings.SplitN(sub.hostname, "*", 2)[0]
	address := fmt.Sprintf("%s:%d", hostname, constants.SubPortNumber)
	sub.lastConnectionStartTime = time.Now()
	srpcClient, err := srpc.DialHTTP("tcp", address)
	if err != nil {
		sub.status = statusFailedToConnect
		return
	}
	defer srpcClient.Close()
	sub.status = statusWaitingToPoll
	sub.lastConnectionSucceededTime = time.Now()
	sub.lastConnectDuration =
		sub.lastConnectionSucceededTime.Sub(sub.lastConnectionStartTime)
	connectDistribution.Add(sub.lastConnectDuration)
	if sub.herd.getImage(sub.requiredImage) == nil {
		sub.status = statusImageNotReady
		return
	}
	sub.herd.pollSemaphore <- true
	sub.status = statusPolling
	sub.poll(srpcClient, previousStatus)
	<-sub.herd.pollSemaphore
}

func (sub *Sub) poll(srpcClient *srpc.Client, previousStatus uint) {
	// If the planned image has just become available, force a full poll.
	if previousStatus == statusSynced &&
		!sub.havePlannedImage &&
		sub.herd.getImage(sub.plannedImage) != nil {
		sub.havePlannedImage = true
		sub.generationCount = 0 // Force a full poll.
	}
	var request subproto.PollRequest
	request.HaveGeneration = sub.generationCount
	var reply subproto.PollResponse
	sub.lastPollStartTime = time.Now()
	logger := sub.herd.logger
	if err := client.CallPoll(srpcClient, request, &reply); err != nil {
		sub.status = statusFailedToPoll
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
	} else {
		if err := fs.RebuildInodePointers(); err != nil {
			sub.status = statusFailedToPoll
			logger.Printf("Error building pointers for: %s %s\n",
				sub.hostname, err)
			return
		}
		fs.BuildInodeToFilenamesTable()
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
	if reply.FetchInProgress {
		sub.status = statusFetching
		return
	}
	if reply.UpdateInProgress {
		sub.status = statusUpdating
		return
	}
	if sub.generationCount < 1 {
		sub.status = statusSubNotReady
		return
	}
	if sub.fileSystem == nil {
		sub.status = previousStatus
		if previousStatus != statusSynced {
			// Force a full poll next cycle so that we can see the state of the
			// sub and recompute/retry what needs to be done.
			sub.generationCount = 0
		}
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
	bool, uint) {
	image := sub.herd.getImage(imageName)
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
		return false, statusFailedToFetch
	}
	return false, statusFetching
}

// Returns true if no update needs to be performed.
func (sub *Sub) sendUpdate(srpcClient *srpc.Client) (bool, uint) {
	logger := sub.herd.logger
	var request subproto.UpdateRequest
	var reply subproto.UpdateResponse
	if sub.buildUpdateRequest(&request) {
		return true, statusSynced
	}
	if err := client.CallUpdate(srpcClient, request, &reply); err != nil {
		logger.Printf("Error calling %s:Subd.Update()\t%s\n", sub.hostname, err)
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
	image := sub.herd.getImage(plannedImageName)
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
	} else {
	}
}
