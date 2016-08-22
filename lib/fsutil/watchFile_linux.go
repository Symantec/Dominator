package fsutil

import (
	"golang.org/x/exp/inotify"
	"io"
	"log"
	"os"
	"path"
	"sync"
)

var (
	lock     sync.RWMutex
	watchers []*inotify.Watcher
)

func watchFileWithInotify(pathname string, channel chan<- io.ReadCloser,
	logger *log.Logger) bool {
	watcher, err := inotify.NewWatcher()
	if err != nil {
		logger.Println("Error creating watcher:", err)
		return false
	}
	lock.Lock()
	defer lock.Unlock()
	watchers = append(watchers, watcher)
	err = watcher.AddWatch(path.Dir(pathname),
		inotify.IN_CREATE|inotify.IN_MOVED_TO)
	if err != nil {
		logger.Println("Error adding watch:", err)
		return false
	}
	go waitForInotifyEvents(watcher, pathname, channel, logger)
	return true
}

func watchFileStopWithInotify() bool {
	lock.Lock()
	defer lock.Unlock()
	// Send cleanup notification to watchers.
	for _, watcher := range watchers {
		watcher.Close()
	}
	// Wait for cleanup of each watcher.
	for _, watcher := range watchers {
		for {
			if _, ok := <-watcher.Event; !ok {
				break
			}
		}
	}
	watchers = nil
	return true
}

func waitForInotifyEvents(watcher *inotify.Watcher, pathname string,
	channel chan<- io.ReadCloser, logger *log.Logger) {
	if file, err := os.Open(pathname); err == nil {
		channel <- file
	}
	for {
		select {
		case event, ok := <-watcher.Event:
			if !ok {
				return
			}
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
		case err, ok := <-watcher.Error:
			if !ok {
				return
			}
			logger.Println("Error with watcher:", err)
		}
	}
}
