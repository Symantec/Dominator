package slavedriver

import (
	"io"
	"net"
	"sync"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
)

type SlaveTrader interface {
	Close() error
	CreateSlave() (SlaveInfo, error)
	DestroySlave(identifier string) error
}

type SlaveDriver struct {
	options     SlaveDriverOptions
	slaveTrader SlaveTrader
	logger      log.DebugLogger
	mutex       sync.Mutex // Lock everthing below (those can change).
	busySlaves  map[*Slave]struct{}
	idleSlaves  map[*Slave]struct{}
	zombies     map[*Slave]struct{}
}

type SlaveDriverOptions struct {
	DatabaseFilename  string
	MaximumIdleSlaves uint
	MinimumIdleSlaves uint
	PortNumber        uint
	Purpose           string
}

type SlaveInfo struct {
	Identifier string
	IpAddress  net.IP
}

type Slave struct {
	clientAddress string
	driver        *SlaveDriver
	info          SlaveInfo
	mutex         sync.Mutex // Lock everthing below (those can change).
	client        *srpc.Client
}

func NewSlaveDriver(options SlaveDriverOptions, slaveTrader SlaveTrader,
	logger log.DebugLogger) (*SlaveDriver, error) {
	return newSlaveDriver(options, slaveTrader, logger)
}

func (driver *SlaveDriver) GetSlave() (*Slave, error) {
	return driver.getSlave()
}

func (driver *SlaveDriver) WriteHtml(writer io.Writer) {
	driver.writeHtml(writer)
}

func (slave *Slave) Destroy() {
	slave.destroy()
}

func (slave *Slave) GetClient() *srpc.Client {
	return slave.getClient()
}

func (slave *Slave) Release() {
	slave.release()
}

func (slave *Slave) String() string {
	return slave.info.Identifier
}
