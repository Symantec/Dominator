package fsutil

import (
	"io"
	"os"
	"path"
	"sync"

	"github.com/Symantec/Dominator/lib/log"
	"gopkg.in/fsnotify/fsnotify.v0"
)

var (
	lock     sync.RWMutex
	watchers []*fsnotify.Watcher
)

func watchFileWithFsNotify(pathname string, channel chan<- io.ReadCloser,
	logger log.Logger) bool {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Println("Error creating watcher:", err)
		return false
	}
	lock.Lock()
	defer lock.Unlock()
	watchers = append(watchers, watcher)
	pathname = path.Clean(pathname)
	dirname := path.Dir(pathname)
	if err := watcher.WatchFlags(dirname, fsnotify.FSN_CREATE); err != nil {
		logger.Println("Error adding watch:", err)
		return false
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
	channel chan<- io.ReadCloser, logger log.Logger) {
	if file, err := os.Open(pathname); err == nil {
		channel <- file
	}
	for {
		select {
		case event, ok := <-watcher.Event:
			if !ok {
				return
			}
			if path.Clean(event.Name) != pathname {
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
