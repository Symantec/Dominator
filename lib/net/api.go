package net

import (
	"net"
	"os"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/nulllogger"
)

const (
	InterfaceTypeBonding = 1 << iota
	InterfaceTypeBridge
	InterfaceTypeEtherNet
	InterfaceTypeVlan
	InterfaceTypeTunTap
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

// BindAndDial is similar to the net.Dial function from the standard library
// except it binds the underlying network socket to a specified local address.
// This gives control over the local port number of the connection, rather than
// a randomly assigned port number.
func BindAndDial(network, localAddr, remoteAddr string, timeout time.Duration) (
	net.Conn, error) {
	return bindAndDial(network, localAddr, remoteAddr, timeout)
}

// CreateTapDevice will create a "tap" network device with a randomly chosen
// interface name. The tap device file and the interface name are returned on
// success, else an error is returned. The device will be destroyed when the
// file is closed.
func CreateTapDevice() (*os.File, string, error) {
	return createTapDevice()
}

// GetBridgeVlanId will get the VLAN Id associated with the uplink EtherNet
// interface for the specified bridge interface. If there is no uplink then the
// returned VLAN Id will be -1.
func GetBridgeVlanId(bridgeIf string) (int, error) {
	return getBridgeVlanId(bridgeIf)
}

// ListBridges will return a list of EtherNet (layer 2) bridges.
func ListBridges() ([]net.Interface, error) {
	bl, _, err := listBroadcastInterfaces(InterfaceTypeBridge, nulllogger.New())
	return bl, err
}

// ListBroadcastInterfaces will return a list of broadcast (EtherNet, bridge,
// vlan) interfaces. Debugging progress messages are logged to logger.
func ListBroadcastInterfaces(interfaceType uint, logger log.DebugLogger) (
	[]net.Interface, map[string]net.Interface, error) {
	return listBroadcastInterfaces(interfaceType, logger)
}

// ListenWithReuse is similar to the net.Listen function from the standard
// library but sets the SO_REUSEADDR and SO_REUSEPORT options on the underlying
// socket. This is needed if using BindAndDial to specify the same local address
// as with the listener.
func ListenWithReuse(network, address string) (net.Listener, error) {
	return listenWithReuse(network, address)
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

// TestCarrier will return true if the interface has a carrier signal.
func TestCarrier(name string) bool {
	return testCarrier(name)
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
