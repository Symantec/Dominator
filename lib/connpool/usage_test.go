package connpool

import (
	"github.com/Symantec/Dominator/lib/resourcepool"
	"net"
	"testing"
)

var serverAddress string

func init() {
	listener, err := net.Listen("tcp", "localhost:")
	if err != nil {
		panic(err)
	}
	serverAddress = listener.Addr().String()
	//go http.Serve(listener, nil)
}

func TestGetUsePut(t *testing.T) {
	cr := New("tcp", serverAddress)
	conn, err := cr.Get(resourcepool.MakeImmediateCanceler(), 0)
	if err != nil {
		t.Error(err)
		return
	}
	conn.LocalAddr()
	conn.Put()
}

func TestGetClosePut(t *testing.T) {
	cr := New("tcp", serverAddress)
	conn, err := cr.Get(resourcepool.MakeImmediateCanceler(), 0)
	if err != nil {
		t.Error(err)
		return
	}
	if err := conn.Close(); err != nil {
		t.Error(err)
	}
	conn.Put()
}

func TestGetPutPut(t *testing.T) {
	cr := New("tcp", serverAddress)
	conn, err := cr.Get(resourcepool.MakeImmediateCanceler(), 0)
	if err != nil {
		t.Error(err)
		return
	}
	conn.Put()
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("Multiple Put() did not panic")
		}
	}()
	conn.Put()
}

func TestUseAfterPut(t *testing.T) {
	cr := New("tcp", serverAddress)
	conn, err := cr.Get(resourcepool.MakeImmediateCanceler(), 0)
	if err != nil {
		t.Error(err)
		return
	}
	conn.Put()
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("Use after Put() did not panic")
		}
	}()
	conn.LocalAddr()
}

func TestUseAfterClose(t *testing.T) {
	cr := New("tcp", serverAddress)
	conn, err := cr.Get(resourcepool.MakeImmediateCanceler(), 0)
	if err != nil {
		t.Error(err)
		return
	}
	conn.Close()
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("Use after Close() did not panic")
		}
	}()
	conn.LocalAddr()
}
