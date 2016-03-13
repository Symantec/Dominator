package fsutil

import (
	"io"
	"log"
	"os"
	"syscall"
	"time"
)

func watchFile(pathname string, logger *log.Logger) <-chan io.Reader {
	channel := make(chan io.Reader, 1)
	go watchFileForever(pathname, channel, logger)
	return channel
}

func watchFileForever(pathname string, channel chan<- io.Reader,
	logger *log.Logger) {
	var lastStat syscall.Stat_t
	var lastFile *os.File
	for ; ; time.Sleep(time.Second) {
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
				channel <- file
				// By holding onto the file, we guarantee that the inode number
				// for the file we've opened cannot be reused until we've seen
				// a new inode.
				if lastFile != nil {
					lastFile.Close()
				}
				lastFile = file
				lastStat = stat
			}
		}
	}
}
