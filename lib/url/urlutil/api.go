package urlutil

import (
	"io"
	"time"

	"github.com/Symantec/Dominator/lib/log"
)

func Open(url string) (io.ReadCloser, error) {
	return open(url)
}

func WatchUrl(url string, checkInterval time.Duration,
	logger log.Logger) (<-chan io.ReadCloser, error) {
	return watchUrl(url, checkInterval, logger)
}
