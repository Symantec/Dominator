// +build !linux

package fsutil

import (
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func watchFileWithFsNotify(pathname string, channel chan<- io.ReadCloser,
	logger log.Logger) bool {
	return false
}

func watchFileStopWithFsNotify() bool { return false }
