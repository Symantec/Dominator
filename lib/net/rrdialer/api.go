/*
	Package rrdialer implements a dialer which provides improved behaviour for
	hostnames with multiple IP addresses (aka. round-robin DNS).

	Unlike the default net.Dialer which divides the timeout between the multiple
	endpoints (IP addresses), the round-robin dialer tracks the historic
	performance of each endpoint to dial the fastest endpoint first and will
	concurrently dial other endpoints if unusually long connection times are
	observed.
*/
package rrdialer

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/log"
)

type Dialer struct {
	dirname   string
	logger    log.DebugLogger
	rawDialer *net.Dialer
	waitGroup sync.WaitGroup
}

// New creates a new Dialer. The underlying raw dialer used to dial each
// endpoint is given by dialer. The directory in which endpoint statitics are
// written is given by cacheDir. If this is the empty string then the ".cache"
// subdirectory of the home directory is used. Log messages are written to
// logger. If the debug level is 3 or greater then all endpoints are dialed.
func New(dialer *net.Dialer, cacheDir string,
	logger log.DebugLogger) (*Dialer, error) {
	return newDialer(dialer, cacheDir, logger)
}

// Dial connects to the address on the named network.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.dialContext(context.Background(), network, address)
}

// DialContext connects to the address on the named network using the provided
// context.
func (d *Dialer) DialContext(ctx context.Context, network,
	address string) (net.Conn, error) {
	return d.dialContext(ctx, network, address)
}

// WaitForBackgroundResults will wait up to timeout for other endpoint
// connection attempts to complete, so that their performance statistics can be
// saved. It is recommended to call this just before the main function returns.
func (d *Dialer) WaitForBackgroundResults(timeout time.Duration) {
	d.waitForBackgroundResults(timeout)
}
