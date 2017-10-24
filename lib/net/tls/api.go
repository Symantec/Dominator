package tls

import (
	"crypto/tls"
	"crypto/x509"
	"net"
)

// Dialer defines a dialer that can be use to create connections.
type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

type TlsDialer struct {
	config *tls.Config
	dialer Dialer
}

// NewDialer creates a Dialer that will use the provided dialer to create TLS
// connections. If the provided dialer is nil the default dialer is used. The
// TLS configuration to use is given by config, which is cloned. If config is
// nil the default configuration is used. A new (TLS) dialer is returned.
func NewDialer(dialer Dialer, config *tls.Config) *TlsDialer {
	return newDialer(dialer, config)
}

// Dial will dial an address and returns a TLS connection.
func (d *TlsDialer) Dial(network, address string) (net.Conn, error) {
	return d.dial(network, address)
}

// NewTestCertificate will return a self-signed certificate for IP address
// 127.0.0.1 that may be used for testing purposes.
func NewTestCertificate() (tls.Certificate, *x509.Certificate, error) {
	return newTestCertificate()
}
