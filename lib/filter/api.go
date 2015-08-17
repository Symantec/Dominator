package filter

import (
	"regexp"
)

type Filter struct {
	FilterLines       []string
	filterExpressions []*regexp.Regexp
}

func (filter *Filter) Compile() error {
	return filter.compile()
}

func (filter *Filter) Match(pathname string) bool {
	return filter.match(pathname)
}
