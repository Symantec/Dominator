package filter

import (
	"regexp"
)

// A Filter contains a list of regular expressions matching pathnames which
// should be filtered out: excluded when building or not changed when pushing
// images to a sub.
// A Filter with no lines is an empty filter (nothing is excluded, everthing is
// changed when pushing).
// A nil *Filter is a sparse filter: when building nothing is excluded. When
// pushing to a sub, all files are pushed but files on the sub which are not in
// the image are not removed from the sub.
type Filter struct {
	FilterLines       []string
	filterExpressions []*regexp.Regexp
}

// A MergeableFilter may be used to combine multiple Filters, eliminating
// duplicate match expressions.
type MergeableFilter struct {
	filterLines map[string]struct{}
}

// Load will load a Filter from a file containing newline separated regular
// expressions.
func Load(filename string) (*Filter, error) {
	return load(filename)
}

// New will create a Filter from a list of regular expressions, which are
// automatically anchored to the beginning of the string to be matched against.
// If filterLines is of length zero the Filter is an empty Filter.
func New(filterLines []string) (*Filter, error) {
	return newFilter(filterLines)
}

// Compile will compile the regular expression strings for later use.
func (filter *Filter) Compile() error {
	return filter.compile()
}

// Match will return true if pathname matches one of the regular expressions.
// The Compile method will be automatically called if the it has not been called
// yet.
func (filter *Filter) Match(pathname string) bool {
	return filter.match(pathname)
}

// ReplaceStrings may be used to replace the regular expression strings with
// de-duplicated copies.
func (filter *Filter) ReplaceStrings(replaceFunc func(string) string) {
	filter.replaceStrings(replaceFunc)
}

// ExportFilter will return a Filter from previously merged Filters.
func (mf *MergeableFilter) ExportFilter() *Filter {
	return mf.exportFilter()
}

// Merge will merge a Filter.
func (mf *MergeableFilter) Merge(filter *Filter) {
	mf.merge(filter)
}
