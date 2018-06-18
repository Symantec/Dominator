package wsyscall

import (
	"os"
	"syscall"
)

func ioctl(fd int, request, argp uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), request,
		argp)
	if errno != 0 {
		return os.NewSyscallError("ioctl", errno)
	}
	return nil
}
