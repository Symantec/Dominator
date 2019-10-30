package rpcclientpool

import (
	gonet "net"
	gorpc "net/rpc"

	"github.com/Cloud-Foundations/Dominator/lib/connpool"
	"github.com/Cloud-Foundations/Dominator/lib/net"
	"github.com/Cloud-Foundations/Dominator/lib/net/rpc"
)

var (
	defaultDialer = &gonet.Dialer{}
)

func newClientResource(network, address string, http bool,
	path string, dialer net.Dialer) *ClientResource {
	if path == "" {
		path = gorpc.DefaultRPCPath
	}
	clientResource := &ClientResource{
		network: network,
		address: address,
		http:    http,
		path:    path,
		dialer:  dialer,
	}
	clientResource.privateClientResource.clientResource = clientResource
	rp := connpool.GetResourcePool()
	clientResource.resource = rp.Create(&clientResource.privateClientResource)
	return clientResource
}

func (cr *ClientResource) get(cancelChannel <-chan struct{}) (*Client, error) {
	if err := cr.resource.Get(cancelChannel); err != nil {
		return nil, err
	}
	cr.client.rpcClient = cr.rpcClient
	return cr.client, nil
}

func (pcr *privateClientResource) Allocate() error {
	cr := pcr.clientResource
	var rpcClient *gorpc.Client
	var err error
	if cr.http {
		rpcClient, err = rpc.DialHTTPPath(
			cr.dialer, cr.network, cr.address, cr.path)
	} else {
		rpcClient, err = rpc.Dial(cr.dialer, cr.network, cr.address)
	}
	if err != nil {
		return err
	}
	client := &Client{resource: cr}
	cr.client = client
	cr.rpcClient = rpcClient
	return nil
}

func (pcr *privateClientResource) Release() error {
	cr := pcr.clientResource
	err := cr.client.rpcClient.Close()
	cr.client.rpcClient = nil
	cr.client = nil
	return err
}

func (client *Client) close() error {
	return client.resource.resource.Release()
}

func (client *Client) put() {
	client.rpcClient = nil
	client.resource.resource.Put()
}
