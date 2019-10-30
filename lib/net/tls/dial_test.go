package tls

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"strings"
	"testing"

	mynet "github.com/Cloud-Foundations/Dominator/lib/net"
)

var (
	serverAddress   string
	tlsCertificate  tls.Certificate
	x509Certificate *x509.Certificate
)

func init() {
	var err error
	tlsCertificate, x509Certificate, err = NewTestCertificate()
	if err != nil {
		panic(err)
	}
	config := &tls.Config{Certificates: []tls.Certificate{tlsCertificate}}
	listener, err := tls.Listen("tcp", "localhost:", config)
	if err != nil {
		panic(err)
	}
	serverAddress = listener.Addr().String()
	go func() {
		for {
			accept(listener)
		}
	}()
}

func accept(listener net.Listener) {
	conn, err := listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	conn.(*tls.Conn).Handshake()
}

func TestDialSkipVerify(t *testing.T) {
	dialer := NewDialer(nil, &tls.Config{InsecureSkipVerify: true})
	if conn, err := dialer.Dial("tcp", serverAddress); err != nil {
		t.Fatal(err)
	} else {
		conn.Close()
	}
}

func TestDialNoConfig(t *testing.T) {
	dialer := NewDialer(nil, nil)
	if conn, err := dialer.Dial("tcp", serverAddress); err != nil {
		if strings.Contains(err.Error(),
			"certificate signed by unknown authority") {
			return
		}
		t.Fatal(err)
	} else {
		conn.Close()
		t.Fatal("Dial with nil config did not fail")
	}
}

func TestDialConfig(t *testing.T) {
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(x509Certificate)
	dialer := NewDialer(nil, &tls.Config{RootCAs: caCertPool})
	if conn, err := dialer.Dial("tcp", serverAddress); err != nil {
		t.Fatal(err)
	} else {
		conn.Close()
	}
}

func TestDialSkipVerifyMeasuringDialer(t *testing.T) {
	dialer := NewDialer(mynet.NewMeasuringDialer(&net.Dialer{}),
		&tls.Config{InsecureSkipVerify: true})
	if conn, err := dialer.Dial("tcp", serverAddress); err != nil {
		t.Fatal(err)
	} else {
		conn.Close()
	}
}
