package urlutil

import (
	"io"
	"time"

	"github.com/Symantec/Dominator/lib/log"
)

// Open will open the URL given by url. A io.ReadCloser is returned, which must
// be closed.
func Open(url string) (io.ReadCloser, error) {
	return open(url)
}

// WatchUrl watches the URL given by url and yields a new io.ReadCloser
// periodically. If the URL is a local file, a new io.ReadCloser is yielded
// when a new inode is found and it is a regular file. If url is a HTTP/HTTPS
// URL a new io.ReadCloser is yielded every checkInterval.
// Each yielded io.ReadCloser must be closed after use.
// Any errors are logged to the logger if it is not nil.
func WatchUrl(url string, checkInterval time.Duration,
	logger log.Logger) (<-chan io.ReadCloser, error) {
	return watchUrl(url, checkInterval, logger)
}
