package fsutil

import (
	"errors"
	"io"
	"os"
	"time"
)

func waitFile(pathname string, timeout time.Duration) (io.ReadCloser, error) {
	stopTime := time.Now().Add(timeout)
	interval := time.Millisecond
	for {
		if file, err := os.Open(pathname); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else {
			return file, nil
		}
		if timeout >= 0 && time.Now().After(stopTime) {
			return nil, errors.New("timed out waiting for file: " + pathname)
		}
		time.Sleep(interval)
		interval *= 2
		if interval > time.Second {
			interval = time.Second
		}
	}
}
