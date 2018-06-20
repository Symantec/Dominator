package net

import (
	"os"
	"strings"
	"syscall"
	"unsafe"

	"github.com/Symantec/Dominator/lib/wsyscall"
)

const (
	cIFF_TUN         = 0x0001
	cIFF_TAP         = 0x0002
	cIFF_NO_PI       = 0x1000
	cIFF_MULTI_QUEUE = 0x0100
)

type ifReq struct {
	Name  [0x10]byte
	Flags uint16
	pad   [0x28 - 0x10 - 2]byte
}

func createTapDevice() (*os.File, string, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return nil, "", err
	}
	req := ifReq{Flags: cIFF_TAP | cIFF_NO_PI}
	err = wsyscall.Ioctl(int(file.Fd()), syscall.TUNSETIFF,
		uintptr(unsafe.Pointer(&req)))
	if err != nil {
		file.Close()
		return nil, "", err
	}
	return file, strings.Trim(string(req.Name[:]), "\x00"), nil
}
