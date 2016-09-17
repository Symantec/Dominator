package rpcclientpool

import (
	"github.com/Symantec/Dominator/lib/connpool"
	"net/rpc"
)

func newClientResource(network, address string, http bool,
	path string) *ClientResource {
	if path == "" {
		path = rpc.DefaultRPCPath
	}
	clientResource := &ClientResource{
		network: network,
		address: address,
		http:    http,
		path:    path,
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
