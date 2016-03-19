// +build !linux

package fsutil

import (
	"io"
	"log"
)

func watchFileWithInotify(pathname string, channel chan<- io.ReadCloser,
	logger *log.Logger) bool {
	return false
}
