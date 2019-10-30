package urlutil

import (
	"io"
	"os"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/log"
)

// Open will open the URL given by url. A io.ReadCloser is returned, which must
// be closed.
func Open(url string) (io.ReadCloser, error) {
	return open(url)
}

type CachedReadCloser struct {
	cache         *os.File
	cacheFilename string
	rawReadCloser io.ReadCloser
}

func (rc *CachedReadCloser) Close() error {
	return rc.close()
}

func (rc *CachedReadCloser) Read(p []byte) (int, error) {
	return rc.read(p)
}

func (rc *CachedReadCloser) SaveCache() error {
	return rc.saveCache()
}

// WatchUrl watches the URL given by url and yields a new io.ReadCloser
// periodically. If the URL is a local file, a new io.ReadCloser is yielded
// when a new inode is found and it is a regular file. If url is a HTTP/HTTPS
// URL a new io.ReadCloser is yielded every checkInterval.
// Each yielded io.ReadCloser must be closed after use.
// Any errors are logged to the logger.
func WatchUrl(url string, checkInterval time.Duration,
	logger log.Logger) (<-chan io.ReadCloser, error) {
	return watchUrl(url, checkInterval, logger)
}

// WatchUrlWithCache is similar to WatchUrl, except that data are cached.
// A cached copy of the data is stored in the file named cacheFilename when the
// SaveCache method is called. This file is read at startup if the URL is not
// available before the initialTimeout.
// Any errors are logged to the logger.
func WatchUrlWithCache(url string, checkInterval time.Duration,
	cacheFilename string, initialTimeout time.Duration,
	logger log.Logger) (<-chan *CachedReadCloser, error) {
	return watchUrlWithCache(url, checkInterval, cacheFilename, initialTimeout,
		logger)
}
