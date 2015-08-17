package filter

import (
	"regexp"
)

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
	for _, regex := range filter.filterExpressions {
		if regex.MatchString(pathname) {
			return true
		}
	}
	return false
}
