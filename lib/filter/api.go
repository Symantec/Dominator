package filter

import (
	"regexp"
)

type Filter struct {
	FilterLines       []string
	filterExpressions []*regexp.Regexp
}

func Load(filename string) (*Filter, error) {
	return load(filename)
}

func New(filterLines []string) (*Filter, error) {
	return newFilter(filterLines)
}

func (filter *Filter) Compile() error {
	return filter.compile()
}

func (filter *Filter) Match(pathname string) bool {
	return filter.match(pathname)
}
