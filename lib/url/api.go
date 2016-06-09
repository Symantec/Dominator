package url

import (
	"net/url"
)

type ParsedQuery struct {
	Flags map[string]struct{}
	Table map[string]string
}

func ParseQuery(URL *url.URL) ParsedQuery {
	return parseQuery(URL)
}
