package smallstack

import (
	"sync"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/slavedriver"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	hyper_proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

type SlaveTrader struct {
	createRequest hyper_proto.CreateVmRequest
	logger        log.DebugLogger
	mutex         sync.Mutex // Lock everything below (those can change).
	hypervisor    *srpc.Client
}

func NewSlaveTrader(createRequest hyper_proto.CreateVmRequest,
	logger log.DebugLogger) (*SlaveTrader, error) {
	return newSlaveTrader(createRequest, logger)
}

func (trader *SlaveTrader) Close() error {
	return trader.close()
}

func (trader *SlaveTrader) CreateSlave() (slavedriver.SlaveInfo, error) {
	return trader.createSlave()
}

func (trader *SlaveTrader) DestroySlave(identifier string) error {
	return trader.destroySlave(identifier)
}
