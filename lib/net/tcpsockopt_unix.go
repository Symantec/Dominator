// +build freebsd linux netbsd

package net

import (
	"syscall"
	"time"
)

func (conn *connection) SetKeepAlivePeriod(d time.Duration) error {
	// The kernel expects seconds so round to next highest second.
	d += (time.Second - time.Nanosecond)
	secs := int(d.Seconds())
	err := syscall.SetsockoptInt(conn.fd, syscall.IPPROTO_TCP,
		syscall.TCP_KEEPINTVL, secs)
	if err != nil {
		return err
	}
	err = syscall.SetsockoptInt(conn.fd, syscall.IPPROTO_TCP,
		syscall.TCP_KEEPIDLE, secs)
	if err != nil {
		return err
	}
	return syscall.SetsockoptInt(conn.fd, syscall.IPPROTO_TCP,
		syscall.TCP_KEEPCNT, 2)
}
