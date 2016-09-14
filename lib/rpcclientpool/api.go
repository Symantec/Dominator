package rpcclientpool

import (
	"github.com/Symantec/Dominator/lib/resourcepool"
	"net/rpc"
)

type Client struct {
	rpcClient *rpc.Client
	resource  *ClientResource
}

func (client *Client) Call(serviceMethod string, args interface{},
	reply interface{}) error {
	return client.rpcClient.Call(serviceMethod, args, reply)
}

func (client *Client) Close() error {
	return client.close()
}

func (client *Client) Go(serviceMethod string, args interface{},
	reply interface{}, done chan *rpc.Call) *rpc.Call {
	return client.rpcClient.Go(serviceMethod, args, reply, done)
}

func (client *Client) Put() {
	client.put()
}

type privateClientResource struct {
	clientResource *ClientResource
}

type ClientResource struct {
	network               string
	address               string
	path                  string
	http                  bool
	resource              *resourcepool.Resource
	privateClientResource privateClientResource
	client                *Client
}

func New(network, address, path string, http bool) *ClientResource {
	return newClientResource(network, address, path, http)
}

func (cr *ClientResource) Get(cancelChannel <-chan struct{}) (*Client, error) {
	return cr.get(cancelChannel)
}
