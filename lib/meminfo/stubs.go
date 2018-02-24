// +build !linux

package meminfo

import "syscall"

func GetMemInfo() (*MemInfo, error) {
	return nil, syscall.ENOTSUPP
}
