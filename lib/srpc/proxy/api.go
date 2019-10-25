package proxy

import (
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

type srpcType struct {
	logger log.DebugLogger
}

func New(logger log.DebugLogger) error {
	return srpc.RegisterNameWithOptions("Proxy", &srpcType{logger: logger},
		srpc.ReceiverOptions{
			PublicMethods: []string{"Connect"}})
}
