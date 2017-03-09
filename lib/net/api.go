package net

import (
	"net"
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
