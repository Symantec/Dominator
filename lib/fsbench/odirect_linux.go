package fsbench

import (
	"os"
	"syscall"
)

func openDirect(name string, flag int, perm os.FileMode) (file *os.File,
	err error) {
	return os.OpenFile(name, flag|syscall.O_DIRECT, perm)
}
