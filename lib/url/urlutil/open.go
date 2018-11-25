package urlutil

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
)

func open(rawurl string) (io.ReadCloser, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "file" {
		return os.Open(u.Path)
	}
	if u.Scheme == "http" || u.Scheme == "https" {
		resp, err := http.Get(rawurl)
		if err != nil {
			return nil,
				errors.New("error getting: " + rawurl + ": " + err.Error())
		}
		if resp.StatusCode != http.StatusOK {
			return nil,
				errors.New("error getting: " + rawurl + ": " + resp.Status)
		}
		return resp.Body, nil
	}
	return nil, errors.New("unknown scheme: " + u.Scheme)
}
