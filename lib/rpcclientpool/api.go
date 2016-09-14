package rpcclientpool

import (
	"github.com/Symantec/Dominator/lib/resourcepool"
	"net/rpc"
)

// Client is a managed RPC client. It implements similar methods as rpc.Client
// from the standard library.
type Client struct {
	rpcClient *rpc.Client
	resource  *ClientResource
}

func (client *Client) Call(serviceMethod string, args interface{},
	reply interface{}) error {
	return client.rpcClient.Call(serviceMethod, args, reply)
}

// Close will close the client, immediately freeing the underlying resource.
// It may no longer be used for communication.
func (client *Client) Close() error {
	return client.close()
}

func (client *Client) Go(serviceMethod string, args interface{},
	reply interface{}, done chan *rpc.Call) *rpc.Call {
	return client.rpcClient.Go(serviceMethod, args, reply, done)
}

// Put will release the client. It may no longer be used for communication.
// It may be internally closed later if required to free limited resources (such
// as file descriptors). If Put is called after Close, no action is taken (this
// is a safe operation and is commonly used in some programming patterns).
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

// New returns a ClientResource for the specified network address. It may be
// used later to obtain a Client which is part of a managed pool of connection
// slots (to limit consumption of resources such as file descriptors). Clients
// can be released with the Put method but the underlying connection may be kept
// open for later re-use. The Client is placed on an internal list.
func New(network, address, path string, http bool) *ClientResource {
	return newClientResource(network, address, path, http)
}

// Get will return a Client. Get will wait until a resource is available or a
// message is received on cancelChannel. If cancelChannel is nil then Get will
// wait indefinitely until a resource is available. If the wait is cancelled
// then Get will return ErrorResourceLimitExceeded.
// Get will panic if it is called again without an intervening Close or Put.
func (cr *ClientResource) Get(cancelChannel <-chan struct{}) (*Client, error) {
	return cr.get(cancelChannel)
}
