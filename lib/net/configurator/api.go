package configurator

import (
	"io"
	"net"

	"github.com/Symantec/Dominator/lib/log"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type bondedInterfaceType struct {
	name   string // "bond0.VlanId" interface name.
	ipAddr net.IP
	subnet *hyper_proto.Subnet
}

type normalInterfaceType struct {
	ipAddr       net.IP
	netInterface net.Interface
	subnet       *hyper_proto.Subnet
}

type NetworkConfig struct {
	bondedInterfaces []bondedInterfaceType
	bridges          []uint
	DefaultSubnet    *hyper_proto.Subnet
	normalInterfaces []normalInterfaceType
	bondSlaves       []string // New interface name.
}

func FindMatchingSubnet(subnets []*hyper_proto.Subnet,
	ipAddr net.IP) *hyper_proto.Subnet {
	return findMatchingSubnet(subnets, ipAddr)
}

func GetNetworkEntries(
	info fm_proto.GetMachineInfoResponse) []fm_proto.NetworkEntry {
	return getNetworkEntries(info)
}

func Compute(machineInfo fm_proto.GetMachineInfoResponse,
	interfaces map[string]net.Interface,
	logger log.DebugLogger) (*NetworkConfig, error) {
	return compute(machineInfo, interfaces, logger)
}

func (netconf *NetworkConfig) PrintDebian(writer io.Writer) error {
	return netconf.printDebian(writer)
}

func (netconf *NetworkConfig) Update(rootDir string,
	logger log.DebugLogger) (bool, error) {
	return netconf.update(rootDir, logger)
}

func (netconf *NetworkConfig) UpdateDebian(rootDir string) (bool, error) {
	return netconf.updateDebian(rootDir)
}

func (netconf *NetworkConfig) WriteDebian(rootDir string) error {
	return netconf.writeDebian(rootDir)
}

func PrintResolvConf(writer io.Writer, subnet *hyper_proto.Subnet) error {
	return printResolvConf(writer, subnet)
}

func UpdateResolvConf(rootDir string,
	subnet *hyper_proto.Subnet) (bool, error) {
	return updateResolvConf(rootDir, subnet)
}

func WriteResolvConf(rootDir string, subnet *hyper_proto.Subnet) error {
	return writeResolvConf(rootDir, subnet)
}
