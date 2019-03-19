package proxy

import (
	"errors"
	"net"
	"net/url"
)

var (
	errorUnsupportedProxy = errors.New("unsupported proxy")
)

func newDialer(proxy string, dialer *net.Dialer) (Dialer, error) {
	if dialer == nil {
		dialer = new(net.Dialer)
	}
	if proxy == "" {
		return dialer, nil
	}
	if parsedProxy, err := url.Parse(proxy); err != nil {
		return nil, err
	} else {
		switch parsedProxy.Scheme {
		case "socks": // Assume SOCKS 5
			return &socksDialer{
				dialer:       dialer,
				proxyAddress: parsedProxy.Host,
				proxyDNS:     true,
				udpSupported: true,
			}, nil
		case "socks4":
			return &socksDialer{
				dialer:       dialer,
				proxyAddress: parsedProxy.Host,
			}, nil
		case "socks4a":
			return &socksDialer{
				dialer:       dialer,
				proxyAddress: parsedProxy.Host,
				proxyDNS:     true,
			}, nil
		case "socks5":
			return &socksDialer{
				dialer:       dialer,
				proxyAddress: parsedProxy.Host,
				proxyDNS:     true,
				udpSupported: true,
			}, nil
		default:
			return nil, errorUnsupportedProxy
		}
	}
}
