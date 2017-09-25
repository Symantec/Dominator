package rpcd

import (
	"encoding/gob"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

var startTime time.Time = time.Now()

func (t *rpcType) Poll(conn *srpc.Conn) error {
	defer conn.Flush()
	var request sub.PollRequest
	var response sub.PollResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	response.NetworkSpeed = t.networkReaderContext.MaximumSpeed()
	response.CurrentConfiguration = t.getConfiguration()
	t.rwLock.RLock()
	response.FetchInProgress = t.fetchInProgress
	response.UpdateInProgress = t.updateInProgress
	if t.lastFetchError != nil {
		response.LastFetchError = t.lastFetchError.Error()
	}
	if !t.updateInProgress {
		if t.lastUpdateError != nil {
			response.LastUpdateError = t.lastUpdateError.Error()
		}
		response.LastUpdateHadTriggerFailures = t.lastUpdateHadTriggerFailures
	}
	response.LastSuccessfulImageName = t.lastSuccessfulImageName
	response.FreeSpace = t.getFreeSpace()
	t.rwLock.RUnlock()
	response.StartTime = startTime
	response.PollTime = time.Now()
	response.ScanCount = t.fileSystemHistory.ScanCount()
	response.DurationOfLastScan = t.fileSystemHistory.DurationOfLastScan()
	response.GenerationCount = t.fileSystemHistory.GenerationCount()
	fs := t.fileSystemHistory.FileSystem()
	if fs != nil &&
		!request.ShortPollOnly &&
		request.HaveGeneration != t.fileSystemHistory.GenerationCount() {
		response.FileSystemFollows = true
	}
	encoder := gob.NewEncoder(conn)
	if err := encoder.Encode(response); err != nil {
		return err
	}
	if response.FileSystemFollows {
		if err := fs.FileSystem.Encode(conn); err != nil {
			return err
		}
		if err := fs.ObjectCache.Encode(conn); err != nil {
			return err
		}
	}
	return nil
}

func (t *rpcType) getFreeSpace() *uint64 {
	if fd, err := syscall.Open(t.rootDir, syscall.O_RDONLY, 0); err != nil {
		t.logger.Printf("error opening: %s: %s", t.rootDir, err)
		return nil
	} else {
		defer syscall.Close(fd)
		var statbuf syscall.Statfs_t
		if err := syscall.Fstatfs(fd, &statbuf); err != nil {
			t.logger.Printf("error getting file-system stats: %s\n", err)
			return nil
		}
		retval := uint64(statbuf.Bfree * uint64(statbuf.Bsize))
		return &retval
	}
}
