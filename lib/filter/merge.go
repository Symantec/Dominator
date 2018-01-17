package filter

import (
	"sort"
)

func (mf *MergeableFilter) exportFilter() *Filter {
	if mf.filterLines == nil {
		return nil // Sparse filter.
	}
	filterLines := make([]string, 0, len(mf.filterLines))
	for filterLine := range mf.filterLines {
		filterLines = append(filterLines, filterLine)
	}
	sort.Strings(filterLines)
	return &Filter{FilterLines: filterLines}
}

func (mf *MergeableFilter) merge(filter *Filter) {
	if filter == nil {
		return // Sparse filter.
	}
	if mf.filterLines == nil {
		mf.filterLines = make(map[string]struct{}, len(filter.FilterLines))
	}
	for _, filterLine := range filter.FilterLines {
		mf.filterLines[filterLine] = struct{}{}
	}
}
