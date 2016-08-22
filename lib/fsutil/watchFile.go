package fsutil

import (
	"io"
	"log"
	"os"
	"syscall"
	"time"
)

var stopped bool

func watchFile(pathname string, logger *log.Logger) <-chan io.ReadCloser {
	channel := make(chan io.ReadCloser, 1)
	if !watchFileWithInotify(pathname, channel, logger) {
		go watchFileForever(pathname, channel, logger)
	}
	return channel
}

func watchFileStop() {
	if !watchFileStopWithInotify() {
		stopped = true
	}
}

func watchFileForever(pathname string, channel chan<- io.ReadCloser,
	logger *log.Logger) {
	var lastStat syscall.Stat_t
	lastFd := -1
	for ; !stopped; time.Sleep(time.Second) {
		var stat syscall.Stat_t
		if err := syscall.Stat(pathname, &stat); err != nil {
			if logger != nil {
				logger.Printf("Error stating file: %s: %s\n", pathname, err)
			}
			continue
		}
		if stat.Ino != lastStat.Ino {
			if file, err := os.Open(pathname); err != nil {
				if logger != nil {
					logger.Printf("Error opening file: %s: %s\n", pathname, err)
				}
				continue
			} else {
				// By holding onto the file, we guarantee that the inode number
				// for the file we've opened cannot be reused until we've seen
				// a new inode.
				if lastFd >= 0 {
					syscall.Close(lastFd)
				}
				lastFd, _ = syscall.Dup(int(file.Fd()))
				channel <- file // Must happen after FD is duplicated.
				lastStat = stat
			}
		}
	}
	if lastFd >= 0 {
		syscall.Close(lastFd)
	}
	close(channel)
}
