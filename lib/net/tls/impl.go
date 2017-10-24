package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"time"
)

func newDialer(dialer Dialer, config *tls.Config) *TlsDialer {
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	if config == nil {
		config = &tls.Config{}
	} else {
		config = config.Clone()
	}
	return &TlsDialer{config, dialer}
}

func (d *TlsDialer) dial(network, address string) (*tls.Conn, error) {
	if dialer, ok := d.dialer.(*net.Dialer); ok {
		return tls.DialWithDialer(dialer, network, address, d.config)
	}
	rawConn, err := d.dialer.Dial(network, address)
	if err != nil {
		return nil, err
	}
	tlsConn := tls.Client(rawConn, d.config)
	if err := tlsConn.Handshake(); err != nil {
		rawConn.Close()
		return nil, err
	}
	return tlsConn, nil
}

func newTestCertificate() (tls.Certificate, *x509.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(time.Minute),
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template,
		&key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	x509Certificate, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	tlsCertificate := tls.Certificate{
		Certificate: [][]byte{x509Certificate.Raw},
		PrivateKey:  key,
	}
	return tlsCertificate, x509Certificate, nil
}
