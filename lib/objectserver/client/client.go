package client

import (
	"errors"
	"fmt"

	"github.com/Symantec/Dominator/lib/srpc"
)

func (objClient *ObjectClient) close() error {
	if objClient.client != nil && objClient.address != "" {
		return objClient.client.Close()
	}
	return nil
}

func (objClient *ObjectClient) getClient() (*srpc.Client, error) {
	if objClient.client != nil {
		return objClient.client, nil
	}
	if objClient.address == "" {
		return nil, errors.New("no client address")
	}
	srpcClient, err := srpc.DialHTTP("tcp", objClient.address, 0)
	if err != nil {
		return nil, fmt.Errorf("error dialing: %s: %s", objClient.address, err)
	}
	objClient.client = srpcClient
	return objClient.client, nil
}
