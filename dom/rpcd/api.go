package rpcd

import (
	"github.com/Symantec/Dominator/dom/herd"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
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
