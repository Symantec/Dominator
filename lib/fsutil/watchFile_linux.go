package fsutil

import (
	"golang.org/x/exp/inotify"
	"io"
	"log"
	"os"
	"path"
)

func watchFileWithInotify(pathname string, channel chan<- io.ReadCloser,
	logger *log.Logger) bool {
	watcher, err := inotify.NewWatcher()
	if err != nil {
		logger.Println("Error creating watcher:", err)
		return false
	}
	err = watcher.AddWatch(path.Dir(pathname),
		inotify.IN_CREATE|inotify.IN_MOVED_TO)
	if err != nil {
		logger.Println("Error adding watch:", err)
		return false
	}
	go waitForInotifyEvents(watcher, pathname, channel, logger)
	return true
}

func waitForInotifyEvents(watcher *inotify.Watcher, pathname string,
	channel chan<- io.ReadCloser, logger *log.Logger) {
	if file, err := os.Open(pathname); err == nil {
		channel <- file
	}
	for {
		select {
		case event := <-watcher.Event:
			if event.Name != pathname {
				continue
			}
			if file, err := os.Open(pathname); err != nil {
				if logger != nil {
					logger.Printf("Error opening file: %s: %s\n", pathname, err)
				}
			} else {
				channel <- file
			}
		case err := <-watcher.Error:
			logger.Println("Error with watcher:", err)
		}
	}
}
