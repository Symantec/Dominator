package srpc

import (
	"crypto/tls"

	"github.com/Cloud-Foundations/Dominator/lib/connpool"
)

func newClientResource(network, address string) *ClientResource {
	clientResource := &ClientResource{
		network: network,
		address: address,
	}
	clientResource.privateClientResource.clientResource = clientResource
	rp := connpool.GetResourcePool()
	clientResource.resource = rp.Create(&clientResource.privateClientResource)
	return clientResource
}

func (cr *ClientResource) getHTTP(tlsConfig *tls.Config,
	cancelChannel <-chan struct{}, dialer connpool.Dialer) (*Client, error) {
	cr.privateClientResource.tlsConfig = tlsConfig
	cr.privateClientResource.dialer = dialer
	if err := cr.resource.Get(cancelChannel); err != nil {
		return nil, err
	}
	cr.inUse = true
	clientMetricsMutex.Lock()
	numInUseClientConnections++
	clientMetricsMutex.Unlock()
	return cr.client, nil
}

func (client *Client) put() {
	client.resource.resource.Put()
	if client.resource.inUse {
		clientMetricsMutex.Lock()
		numInUseClientConnections--
		clientMetricsMutex.Unlock()
		client.resource.inUse = false
	}
}

func (pcr *privateClientResource) Allocate() error {
	cr := pcr.clientResource
	client, err := dialHTTP(cr.network, cr.address, pcr.tlsConfig, pcr.dialer)
	if err != nil {
		return err
	}
	cr.client = client
	client.resource = cr
	return nil
}

func (pcr *privateClientResource) Release() error {
	cr := pcr.clientResource
	err := cr.client.conn.Close()
	cr.client = nil
	return err
}
