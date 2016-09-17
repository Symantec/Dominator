package rpcclientpool

import (
	"github.com/Symantec/Dominator/lib/resourcepool"
	"net"
	"net/http"
	"net/rpc"
	"testing"
)

var serverAddress string

func init() {
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", "localhost:")
	if err != nil {
		panic(err)
	}
	serverAddress = listener.Addr().String()
	go http.Serve(listener, nil)
}

func TestGetCallPut(t *testing.T) {
	cr := New("tcp", serverAddress, true, "")
	client, err := cr.Get(resourcepool.MakeImmediateCanceler())
	if err != nil {
		t.Error(err)
		return
	}
	var request, reply int
	client.Call("Service.Method", request, &reply)
	client.Put()
}

func TestGetClosePut(t *testing.T) {
	cr := New("tcp", serverAddress, true, "")
	client, err := cr.Get(resourcepool.MakeImmediateCanceler())
	if err != nil {
		t.Error(err)
		return
	}
	if err := client.Close(); err != nil {
		t.Error(err)
	}
	client.Put()
}

func TestGetPutPut(t *testing.T) {
	cr := New("tcp", serverAddress, true, "")
	client, err := cr.Get(resourcepool.MakeImmediateCanceler())
	if err != nil {
		t.Error(err)
		return
	}
	client.Put()
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("Multiple Put() did not panic")
		}
	}()
	client.Put()
}

func TestCallAfterPut(t *testing.T) {
	cr := New("tcp", serverAddress, true, "")
	client, err := cr.Get(resourcepool.MakeImmediateCanceler())
	if err != nil {
		t.Error(err)
		return
	}
	client.Put()
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("Call() after Put() did not panic")
		}
	}()
	var request, reply int
	err = client.Call("Service.Method", request, &reply)
	t.Error(err)
}
