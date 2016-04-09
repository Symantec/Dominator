package verstr

import (
	sortlib "sort"
)

type sliceWrapper []string

func (list sliceWrapper) Len() int {
	return len(list)
}

func (list sliceWrapper) Less(i, j int) bool {
	return Less(list[i], list[j])
}

func (list sliceWrapper) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func sort(list []string) {
	sortlib.Sort(sliceWrapper(list))
}
