package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/dom/herd"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

type rpcType struct {
	herd   *herd.Herd
	logger log.Logger
}

func Setup(herd *herd.Herd, logger log.Logger) {
	rpcObj := &rpcType{
		herd:   herd,
		logger: logger}
	srpc.RegisterName("Dominator", rpcObj)
}
