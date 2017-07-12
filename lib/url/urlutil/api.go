package urlutil

import "io"

func Open(url string) (io.ReadCloser, error) {
	return open(url)
}
