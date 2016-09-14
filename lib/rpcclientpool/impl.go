package rpcclientpool

import (
	"github.com/Symantec/Dominator/lib/connpool"
	"net/rpc"
)

func newClientResource(network, address, path string,
	http bool) *ClientResource {
	if path == "" {
		path = rpc.DefaultRPCPath
	}
	clientResource := &ClientResource{
		network: network,
		address: address,
		path:    path,
		http:    http,
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
	return cr.client, nil
}

func (pcr *privateClientResource) Allocate() error {
	cr := pcr.clientResource
	var rpcClient *rpc.Client
	var err error
	if cr.http {
		rpcClient, err = rpc.DialHTTPPath(cr.network, cr.address, cr.path)
	} else {
		rpcClient, err = rpc.Dial(cr.network, cr.address)
	}
	if err != nil {
		return err
	}
	client := &Client{rpcClient: rpcClient, resource: cr}
	cr.client = client
	client.resource = cr
	return nil
}

func (pcr *privateClientResource) Release() error {
	cr := pcr.clientResource
	err := cr.client.rpcClient.Close()
	cr.client = nil
	return err
}

func (client *Client) close() error {
	return client.resource.resource.Release()
}

func (client *Client) put() {
	client.resource.resource.Put()
}
