package urlutil

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/log"
)

func watchUrl(rawurl string, checkInterval time.Duration,
	logger log.Logger) (<-chan io.ReadCloser, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "file" {
		return fsutil.WatchFile(u.Path, logger), nil
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		ch := make(chan io.ReadCloser, 1)
		go watchUrlLoop(rawurl, checkInterval, ch, logger)
		return ch, nil
	}
	return nil, errors.New("unknown scheme: " + u.Scheme)
}

func watchUrlLoop(rawurl string, checkInterval time.Duration,
	ch chan<- io.ReadCloser, logger log.Logger) {
	for ; ; time.Sleep(checkInterval) {
		resp, err := http.Get(rawurl)
		if err != nil {
			logger.Println(err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			logger.Println(resp.Status)
			continue
		}
		ch <- resp.Body
	}
}
