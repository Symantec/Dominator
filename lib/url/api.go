package url

import (
	"net/url"
)

const (
	OutputTypeHtml = iota
	OutputTypeText
	OutputTypeJson
)

type ParsedQuery struct {
	Flags map[string]struct{}
	Table map[string]string
}

func ParseQuery(URL *url.URL) ParsedQuery {
	return parseQuery(URL)
}

func (pq ParsedQuery) OutputType() uint {
	return pq.outputType()
}
