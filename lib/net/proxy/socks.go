package proxy

import (
	"errors"
	"io"
	"net"
	"strconv"
)

var (
	errorNoIPs                = errors.New("no IP addresses")
	errorNoIPv4s              = errors.New("no IPv4 addresses")
	errorNotIPv4              = errors.New("not an IPv4 address")
	errorShortRead            = errors.New("short read")
	errorShortWrite           = errors.New("short write")
	errorUnsupportedTransport = errors.New("unsupported transport")

	errorRequestRejected     = errors.New("request rejected")
	errorNoIdentd            = errors.New("no client identd")
	errorUserIdNotConfirmed  = errors.New("user ID not confirmed")
	errorUnknownResponseCode = errors.New("unknown response code")
)

type socksDialer struct {
	dialer       *net.Dialer
	proxyAddress string
	proxyDNS     bool
	udpSupported bool
}

func (d *socksDialer) Dial(network, address string) (net.Conn, error) {
	switch network {
	case "tcp":
		return d.dialTCP(address)
	case "udp":
	}
	return nil, errorUnsupportedTransport
}

func (d *socksDialer) dialTCP(address string) (net.Conn, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, err
	}
	var request []byte
	if ip := net.ParseIP(host); ip != nil {
		if request, err = makeSocks4IpRequest(ip, uint16(port)); err != nil {
			return nil, err
		}
	} else if d.proxyDNS {
		if request, err = makeSocks4aRequest(host, uint16(port)); err != nil {
			return nil, err
		}
	} else {
		if request, err = makeSocks4Request(host, uint16(port)); err != nil {
			return nil, err
		}
	}
	if conn, err := d.dialer.Dial("tcp", d.proxyAddress); err != nil {
		return nil, err
	} else {
		if nWritten, err := conn.Write(request); err != nil {
			conn.Close()
			return nil, err
		} else if nWritten < len(request) {
			conn.Close()
			return nil, errorShortWrite
		}
		if err := readSocks4Response(conn); err != nil {
			conn.Close()
			return nil, err
		}
		return conn, nil
	}
}

func makeSocks4aRequest(host string, port uint16) ([]byte, error) {
	request := make([]byte, 10+len(host))
	request[0] = 0x04
	request[1] = 0x01
	request[2] = byte(port >> 8)
	request[3] = byte(port & 0xff)
	request[7] = 0xff
	copy(request[9:], host)
	return request, nil
}

func makeSocks4IpRequest(ip net.IP, port uint16) ([]byte, error) {
	if ip = ip.To4(); ip == nil {
		return nil, errorNoIPv4s
	}
	request := make([]byte, 9)
	request[0] = 0x04
	request[1] = 0x01
	request[2] = byte(port >> 8)
	request[3] = byte(port & 0xff)
	request[4] = ip[0]
	request[5] = ip[1]
	request[6] = ip[2]
	request[7] = ip[3]
	return request, nil
}

func makeSocks4Request(host string, port uint16) ([]byte, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	if len(ips) < 1 {
		return nil, errorNoIPs
	}
	var ip4 net.IP
	for _, ip := range ips {
		if ip4 = ip.To4(); ip4 != nil {
			break
		}
	}
	if len(ip4) != 4 {
		return nil, errorNoIPv4s
	}
	return makeSocks4IpRequest(ip4, port)
}

func readSocks4Response(reader io.Reader) error {
	response := make([]byte, 8)
	if nRead, err := reader.Read(response); err != nil {
		return err
	} else if nRead < len(response) {
		return errorShortRead
	} else {
		switch response[1] {
		case 0x5a:
			return nil
		case 0x5b:
			return errorRequestRejected
		case 0x5c:
			return errorNoIdentd
		case 0x5d:
			return errorUserIdNotConfirmed
		default:
			return errorUnknownResponseCode
		}
	}
}
