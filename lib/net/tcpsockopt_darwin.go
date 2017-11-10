package net

import (
	"syscall"
	"time"
)

const sysTCP_KEEPINTVL = 0x101

func (conn *connection) SetKeepAlivePeriod(d time.Duration) error {
	// The kernel expects seconds so round to next highest second.
	d += (time.Second - time.Nanosecond)
	secs := int(d.Seconds())
	err := syscall.SetsockoptInt(conn.fd, syscall.IPPROTO_TCP,
		sysTCP_KEEPINTVL, secs)
	if err != nil {
		return err
	}
	return syscall.SetsockoptInt(conn.fd, syscall.IPPROTO_TCP,
		syscall.TCP_KEEPALIVE, 2)
}
