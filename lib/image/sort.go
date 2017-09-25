package image

import (
	"sort"

	"github.com/Symantec/Dominator/lib/verstr"
)

type directoryList []Directory

func (list directoryList) Len() int {
	return len(list)
}

func (list directoryList) Less(i, j int) bool {
	return verstr.Less(list[i].Name, list[j].Name)
}

func (list directoryList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func sortDirectories(directories []Directory) {
	sort.Sort(directoryList(directories))
}
