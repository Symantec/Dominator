package srpc

import (
	"crypto/tls"
	"github.com/Symantec/Dominator/lib/connpool"
	"time"
)

func newClientResource(network, address string) *ClientResource {
	return &ClientResource{
		network:  network,
		address:  address,
		resource: connpool.GetResourcePool().Create(),
	}
}

func (cr *ClientResource) getHTTP(tlsConfig *tls.Config, wait bool,
	timeout time.Duration) (*Client, error) {
	if !cr.resource.Get(wait) {
		return nil, connpool.ErrorResourceLimitExceeded
	}
	if cr.client != nil {
		return cr.client, nil
	}
	client, err := dialHTTP(cr.network, cr.address, tlsConfig, timeout)
	if err != nil {
		cr.resource.Put() // Free up a slot for someone else.
		return nil, err
	}
	cr.client = client
	client.resource = cr
	cr.resource.SetReleaseFunc(cr.releaseCallback)
	return client, nil
}

func (cr *ClientResource) releaseCallback() {
	cr.closeError = cr.client.conn.Close()
	cr.client = nil
}

func (client *Client) put() {
	client.resource.resource.Put()
}
