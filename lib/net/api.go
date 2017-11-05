package net

import (
	"net"
	"time"
)

// CpuSharer defines the interface for types which can be used to
// co-operatively share CPUs.
type CpuSharer interface {
	GrabCpu()
	ReleaseCpu()
}

// Dialer defines a dialer that can be use to create connections.
type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

type TCPConn interface {
	net.Conn
	SetKeepAlive(keepalive bool) error
	SetKeepAlivePeriod(d time.Duration) error
}

// NewCpuSharingDialer wraps dialer and returns a new Dialer which uses the
// cpuSharer to limit concurrent CPU usage.
// Whenever a blocking operation is about to commence (such a Dial or Read or
// Write for the connection) the cpuSharer.ReleaseCpu method is called, allowing
// the blocking operation to run concurrently.
// When the blocking operation completes, the cpuSharer.GrabCpu method is
// called, limiting the number of running goroutines that have to compete for
// the CPU.
// This helps avoid the thundering herd problem where large numbers of
// goroutines processing network transactions overwhelm the CPU and affect the
// responsiveness of other goroutines such as dashboards and health checks.
func NewCpuSharingDialer(dialer Dialer, cpuSharer CpuSharer) Dialer {
	return newCpuSharingDialer(dialer, cpuSharer)
}

// MeasuringConnection implements the net.Conn interface. Additionally it has
// methods to return the cumulative time spent blocking in the Read or Write
// methods.
type MeasuringConnection struct {
	net.Conn
	cumulativeReadTime  time.Duration
	cumulativeWriteTime time.Duration
}

// CumulativeReadTime returns the cumulative time spent blocking on Read.
func (conn *MeasuringConnection) CumulativeReadTime() time.Duration {
	return conn.cumulativeReadTime
}

// CumulativeReadTime returns the cumulative time spent blocking on Write.
func (conn *MeasuringConnection) CumulativeWriteTime() time.Duration {
	return conn.cumulativeWriteTime
}

func (conn *MeasuringConnection) Read(b []byte) (n int, err error) {
	return conn.read(b)
}

func (conn *MeasuringConnection) Write(b []byte) (n int, err error) {
	return conn.write(b)
}

// MeasuringDialer implements the Dialer interface. Additionally it has
// methods to return the cumulative time spent blocking in the Dial method.
type MeasuringDialer struct {
	dialer             Dialer
	cumulativeDialTime time.Duration
}

// NewMeasuringDialer wraps dialer and returns a dialer that can be used to
// measure the time spent in blocking operations.
func NewMeasuringDialer(dialer Dialer) *MeasuringDialer {
	return newMeasuringDialer(dialer)
}

// CumulativeDialTime returns the cumulative time spent blocking on Dial.
func (d *MeasuringDialer) CumulativeDialTime() time.Duration {
	return d.cumulativeDialTime
}
