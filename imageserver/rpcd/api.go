package rpcd

import (
	"net/rpc"
)

type Imageserver int

func Setup() {
	imageserver := new(Imageserver)
	rpc.Register(imageserver)
	rpc.HandleHTTP()
}
