// +build !linux

package fsbench

import (
	"os"
)

func openDirect(name string, flag int, perm os.FileMode) (file *os.File,
	err error) {
	return os.OpenFile(name, flag, perm)
}
