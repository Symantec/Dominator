// +build !linux

package meminfo

import "syscall"

func getMemInfo() (*MemInfo, error) {
	return nil, syscall.ENOTSUP
}
