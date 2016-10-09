package fsutil

import (
	"gopkg.in/fsnotify.v0"
	"io"
	"log"
	"os"
	"path"
	"sync"
)

var (
	lock     sync.RWMutex
	watchers []*fsnotify.Watcher
)

func watchFileWithFsNotify(pathname string, channel chan<- io.ReadCloser,
	logger *log.Logger) bool {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Println("Error creating watcher:", err)
		return false
	}
	lock.Lock()
	defer lock.Unlock()
	watchers = append(watchers, watcher)
	if err := watcher.Watch(path.Dir(pathname)); err != nil {
		logger.Println("Error adding watch:", err)
		return false
	}
	if err := watcher.WatchFlags(pathname,
		fsnotify.FSN_CREATE|fsnotify.FSN_RENAME); err != nil {
		logger.Println("Error setting flags:", err)
	}
	go waitForNotifyEvents(watcher, pathname, channel, logger)
	return true
}

func watchFileStopWithFsNotify() bool {
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

func waitForNotifyEvents(watcher *fsnotify.Watcher, pathname string,
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
				if os.IsNotExist(err) {
					continue
				}
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
