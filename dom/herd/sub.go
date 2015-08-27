package herd

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/hash"
	subproto "github.com/Symantec/Dominator/proto/sub"
	"net/rpc"
	"strings"
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

func (sub *Sub) disconnect() {
	sub.connection.Close()
	sub.connection = nil
}

func (sub *Sub) connectAndPoll() {
	hostname := strings.SplitN(sub.hostname, "*", 2)[0]
	var err error
	sub.connection, err = rpc.DialHTTP("tcp",
		fmt.Sprintf("%s:%d", hostname, constants.SubPortNumber))
	if err != nil {
		fmt.Printf("Error dialing\t%s\n", err)
		return
	}
	defer sub.disconnect()
	sub.herd.pollSemaphore <- true
	sub.poll()
	<-sub.herd.pollSemaphore
}

func (sub *Sub) poll() {
	var request subproto.PollRequest
	request.HaveGeneration = sub.generationCount
	var reply subproto.PollResponse
	err := sub.connection.Call("Subd.Poll", request, &reply)
	if err != nil {
		fmt.Printf("Error calling\t%s\n", err)
		return
	}
	fs := reply.FileSystem
	if fs != nil {
		fs.RebuildPointers()
		sub.fileSystem = fs
		sub.generationCount = reply.GenerationCount
		fmt.Printf("Polled: %s, GenerationCount=%d\n",
			sub.hostname, reply.GenerationCount)
	}
	if reply.FetchInProgress || reply.UpdateInProgress {
		return
	}
	if sub.generationCountAtChangeStart == sub.generationCount {
		return
	}
	if !sub.fetchMissingObjects(sub.requiredImage) {
		return
	}
	if !sub.sendUpdate() {
		return
	}
	sub.fetchMissingObjects(sub.plannedImage)
}

// Returns true if all required objects are available.
func (sub *Sub) fetchMissingObjects(imageName string) bool {
	if sub.fileSystem == nil {
		return false
	}
	if imageName == "" {
		return false
	}
	image := sub.herd.getImage(imageName)
	if image == nil {
		return false
	}
	missingObjects := make(map[hash.Hash]bool)
	for _, inode := range image.FileSystem.RegularInodeTable {
		if inode.Size > 0 {
			missingObjects[inode.Hash] = true
		}
	}
	for _, hash := range sub.fileSystem.ObjectCache {
		delete(missingObjects, hash)
	}
	for _, inode := range sub.fileSystem.RegularInodeTable {
		if inode.Size > 0 {
			delete(missingObjects, inode.Hash)
		}
	}
	if len(missingObjects) < 1 {
		return true
	}
	// TODO(rgooch): Remove debugging output.
	fmt.Printf("Objects needing to be fetched: %d\n", len(missingObjects))
	var request subproto.FetchRequest
	var reply subproto.FetchResponse
	request.ServerAddress = sub.herd.imageServerAddress
	for hash, _ := range missingObjects {
		request.Hashes = append(request.Hashes, hash)
	}
	err := sub.connection.Call("Subd.Fetch", request, &reply)
	if err != nil {
		fmt.Printf("Error calling\t%s\n", err)
		return false
	}
	sub.generationCountAtChangeStart = sub.generationCount
	return false
}

// Returns true if no update needs to be performed.
func (sub *Sub) sendUpdate() bool {
	// TODO(rgooch): Implement this.
	return false
}
