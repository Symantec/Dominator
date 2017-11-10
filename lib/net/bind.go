package net

import (
	"errors"
	"io"
	"net"
	"sync"
	"syscall"
	"time"
)

const SO_REUSEPORT = 15

var errorTimeout = errors.New("timeout")

type connection struct {
	fd            int
	localAddress  *net.TCPAddr
	remoteAddress *net.TCPAddr
	lock          sync.Mutex
	deadline      time.Time
	readDeadline  time.Time
	writeDeadline time.Time
}

func bindAndDial(network, localAddr, remoteAddr string, timeout time.Duration) (
	net.Conn, error) {
	if network != "tcp" && network != "tcp4" {
		return net.DialTimeout(network, remoteAddr, timeout)
	}
	sockFd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return nil, errors.New("error creating socket: " + err.Error())
	}
	defer func() {
		if sockFd >= 0 {
			syscall.Close(sockFd)
		}
	}()
	if err := setReuse(sockFd); err != nil {
		return nil, err
	}
	if err := setReadTimeout(sockFd, timeout); err != nil {
		return nil, err
	}
	if err := setWriteTimeout(sockFd, timeout); err != nil {
		return nil, err
	}
	localTCPAddr, localSockAddr, err := resolveAddr(localAddr)
	if err != nil {
		return nil, err
	}
	if err := syscall.Bind(sockFd, localSockAddr); err != nil {
		return nil, errors.New("error binding to: " + localAddr + " : " +
			err.Error())
	}
	remTCPAddr, remSockAddr, err := resolveAddr(remoteAddr)
	if err != nil {
		return nil, err
	}
	if err := syscall.Connect(sockFd, remSockAddr); err != nil {
		return nil, errors.New("error connecting to: " + remoteAddr + " : " +
			err.Error())
	}
	if err := setReadTimeout(sockFd, 0); err != nil {
		return nil, err
	}
	if err := setWriteTimeout(sockFd, 0); err != nil {
		return nil, err
	}
	conn := &connection{
		fd:            sockFd,
		localAddress:  localTCPAddr,
		remoteAddress: remTCPAddr,
	}
	sockFd = -1 // Prevent Close on return.
	return conn, nil
}

func listenWithReuse(network, address string) (net.Listener, error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	if tcpListener, ok := listener.(*net.TCPListener); ok {
		file, err := tcpListener.File()
		if err != nil {
			return nil, err
		}
		defer file.Close()
		if err := setReuse(int(file.Fd())); err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("not a TCPlistener")
	}
	return listener, nil
}

func read(fd int, b []byte) (int, error) {
	if nRead, err := syscall.Read(fd, b); err != nil {
		return 0, err
	} else if nRead <= 0 {
		return 0, io.EOF
	} else {
		return nRead, nil
	}
}

func resolveAddr(address string) (*net.TCPAddr, *syscall.SockaddrInet4, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, nil, err
	}
	if len(tcpAddr.IP) < 1 {
		return tcpAddr, &syscall.SockaddrInet4{Port: tcpAddr.Port}, nil
	}
	tcp4IP := tcpAddr.IP.To4()
	if tcp4IP == nil {
		return nil, nil, errors.New(address + " is not an IPv4 address")
	}
	var ip4 [4]byte
	for index, b := range tcp4IP {
		ip4[index] = b
	}
	return tcpAddr, &syscall.SockaddrInet4{Port: tcpAddr.Port, Addr: ip4}, nil
}

func setReadTimeout(fd int, timeout time.Duration) error {
	timeval := syscall.NsecToTimeval(timeout.Nanoseconds())
	return syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET,
		syscall.SO_RCVTIMEO, &timeval)
}

func setReuse(fd int) error {
	err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR,
		1)
	if err != nil {
		return err
	}
	return syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, SO_REUSEPORT, 1)
}

func setWriteTimeout(fd int, timeout time.Duration) error {
	timeval := syscall.NsecToTimeval(timeout.Nanoseconds())
	return syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET,
		syscall.SO_SNDTIMEO, &timeval)
}

func write(fd int, b []byte) (int, error) {
	if nWritten, err := syscall.Write(fd, b); err != nil {
		return 0, err
	} else if nWritten < len(b) {
		return nWritten, io.EOF
	} else {
		return nWritten, nil
	}
}

func (conn *connection) Close() error {
	return syscall.Close(conn.fd)
}

func (conn *connection) getDeadline(write bool) time.Time {
	conn.lock.Lock()
	defer conn.lock.Unlock()
	deadline := conn.readDeadline
	if write {
		deadline = conn.writeDeadline
	}
	if conn.deadline.IsZero() {
		return deadline
	} else if !deadline.IsZero() && deadline.Before(conn.deadline) {
		return deadline
	} else {
		return conn.deadline
	}
}

func (conn *connection) LocalAddr() net.Addr {
	return conn.localAddress
}

func (conn *connection) Read(b []byte) (int, error) {
	deadline := conn.getDeadline(false)
	if deadline.IsZero() { // Fast check.
		return read(conn.fd, b)
	}
	timeout := time.Until(deadline)
	if timeout <= 0 {
		return 0, errorTimeout
	}
	if err := setReadTimeout(conn.fd, timeout); err != nil {
		return 0, err
	}
	nRead, err := read(conn.fd, b)
	if err == syscall.EAGAIN {
		err = errorTimeout
	}
	if e := setReadTimeout(conn.fd, 0); err == nil {
		err = e
	}
	if err != nil {
		return 0, err
	}
	return nRead, nil
}

func (conn *connection) RemoteAddr() net.Addr {
	return conn.remoteAddress
}

func (conn *connection) SetDeadline(t time.Time) error {
	conn.lock.Lock()
	defer conn.lock.Unlock()
	conn.deadline = t
	return nil
}

func (conn *connection) SetKeepAlive(keepalive bool) error {
	var ka int
	if keepalive {
		ka = 1
	}
	return syscall.SetsockoptInt(conn.fd, syscall.SOL_SOCKET,
		syscall.SO_KEEPALIVE, ka)
}

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

func (conn *connection) SetReadDeadline(t time.Time) error {
	conn.lock.Lock()
	defer conn.lock.Unlock()
	conn.readDeadline = t
	return nil
}

func (conn *connection) SetWriteDeadline(t time.Time) error {
	conn.lock.Lock()
	defer conn.lock.Unlock()
	conn.writeDeadline = t
	return nil
}

func (conn *connection) Write(b []byte) (int, error) {
	deadline := conn.getDeadline(true)
	if deadline.IsZero() { // Fast check.
		return write(conn.fd, b)
	}
	timeout := time.Until(deadline)
	if timeout <= 0 {
		return 0, errorTimeout
	}
	if err := setWriteTimeout(conn.fd, timeout); err != nil {
		return 0, err
	}
	nWritten, err := write(conn.fd, b)
	if err == syscall.EAGAIN {
		err = errorTimeout
	}
	if e := setWriteTimeout(conn.fd, 0); err == nil {
		err = e
	}
	if err != nil {
		return 0, err
	}
	return nWritten, nil
}
