package urlutil

import (
	"io"
	"os"
	"syscall"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/log"
)

const (
	privateFilePerms = syscall.S_IRUSR | syscall.S_IWUSR
)

func handleReadClosers(cachedReadCloserChannel chan<- *CachedReadCloser,
	readCloserChannel <-chan io.ReadCloser, cacheFilename string,
	initialTimeout time.Duration, logger log.Logger) {
	timer := time.NewTimer(initialTimeout)
	select {
	case <-timer.C:
		if file, err := os.Open(cacheFilename); err == nil {
			cachedReadCloserChannel <- &CachedReadCloser{rawReadCloser: file}
		}
	case readCloser := <-readCloserChannel:
		timer.Stop()
		cachedReadCloserChannel <- newCachedReadCloser(cacheFilename,
			readCloser, logger)
	}
	for readCloser := range readCloserChannel {
		cachedReadCloserChannel <- newCachedReadCloser(cacheFilename,
			readCloser, logger)
	}
}

func newCachedReadCloser(cacheFilename string,
	readCloser io.ReadCloser, logger log.Logger) *CachedReadCloser {
	file, err := os.OpenFile(cacheFilename+"~", os.O_CREATE|os.O_WRONLY,
		privateFilePerms)
	if err != nil {
		logger.Println(err)
	}
	rc := &CachedReadCloser{
		cache:         file,
		cacheFilename: cacheFilename,
		rawReadCloser: readCloser,
	}
	return rc
}

func (rc *CachedReadCloser) close() error {
	if rc.cache != nil {
		rc.cache.Close()
		os.Remove(rc.cacheFilename + "~")
	}
	return rc.rawReadCloser.Close()
}

func (rc *CachedReadCloser) read(p []byte) (int, error) {
	if rc.cache == nil {
		return rc.rawReadCloser.Read(p)
	}
	if nRead, err := rc.rawReadCloser.Read(p); err != nil && err != io.EOF {
		rc.cache.Close()
		rc.cache = nil
		os.Remove(rc.cacheFilename + "~")
		return nRead, err
	} else {
		if nWritten, err := rc.cache.Write(p[:nRead]); err != nil {
			rc.cache.Close()
			rc.cache = nil
			os.Remove(rc.cacheFilename + "~")
		} else if nWritten < nRead {
			rc.cache.Close()
			rc.cache = nil
			os.Remove(rc.cacheFilename + "~")
		}
		return nRead, err
	}
}

func (rc *CachedReadCloser) saveCache() error {
	if rc.cache == nil {
		return nil
	}
	err := rc.cache.Close()
	rc.cache = nil
	if err != nil {
		os.Remove(rc.cacheFilename + "~")
		return err
	}
	return os.Rename(rc.cacheFilename+"~", rc.cacheFilename)
}

func watchUrlWithCache(rawurl string, checkInterval time.Duration,
	cacheFilename string, initialTimeout time.Duration,
	logger log.Logger) (<-chan *CachedReadCloser, error) {
	readCloserChannel, err := watchUrl(rawurl, checkInterval, logger)
	if err != nil {
		return nil, err
	}
	cachedReadCloserChannel := make(chan *CachedReadCloser, 1)
	go handleReadClosers(cachedReadCloserChannel, readCloserChannel,
		cacheFilename, initialTimeout, logger)
	return cachedReadCloserChannel, nil
}
