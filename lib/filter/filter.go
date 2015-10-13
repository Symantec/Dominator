package filter

import (
	"regexp"
)

func newFilter(filterLines []string) (*Filter, error) {
	var filter Filter
	filter.FilterLines = filterLines
	if err := filter.compile(); err != nil {
		return nil, err
	}
	return &filter, nil
}

func (filter *Filter) compile() error {
	filter.filterExpressions = make([]*regexp.Regexp, len(filter.FilterLines))
	for index, reEntry := range filter.FilterLines {
		var err error
		filter.filterExpressions[index], err = regexp.Compile("^" + reEntry)
		if err != nil {
			return err
		}
	}
	return nil
}

func (filter *Filter) match(pathname string) bool {
	if len(filter.filterExpressions) != len(filter.FilterLines) {
		filter.compile()
	}
	for _, regex := range filter.filterExpressions {
		if regex.MatchString(pathname) {
			return true
		}
	}
	return false
}
