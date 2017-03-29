// +build !linux

package fsutil

import (
	"github.com/Symantec/Dominator/lib/log"
	"io"
)

func watchFileWithFsNotify(pathname string, channel chan<- io.ReadCloser,
	logger log.Logger) bool {
	return false
}

func watchFileStopWithFsNotify() bool { return false }
