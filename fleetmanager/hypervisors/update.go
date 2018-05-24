package hypervisors

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/Symantec/Dominator/fleetmanager/topology"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

var (
	desiredAddressPoolSize = flag.Uint("desiredAddressPoolSize", 16,
		"Desired number of free addresses to maintain in Hypervisor")
	maximumAddressPoolSize = flag.Uint("maximumAddressPoolSize", 24,
		"Maximum number of free addresses to maintain in Hypervisor")
	minimumAddressPoolSize = flag.Uint("minimumAddressPoolSize", 8,
		"Minimum number of free addresses to maintain in Hypervisor")
)

func checkPoolLimits() error {
	if *desiredAddressPoolSize < *minimumAddressPoolSize {
		return fmt.Errorf(
			"desiredAddressPoolSize: %d is less than minimumAddressPoolSize: %d",
			*desiredAddressPoolSize, *minimumAddressPoolSize)
	}
	if *desiredAddressPoolSize > *maximumAddressPoolSize {
		return fmt.Errorf(
			"desiredAddressPoolSize: %d is greater than maximumAddressPoolSize: %d",
			*desiredAddressPoolSize, *maximumAddressPoolSize)
	}
	return nil
}

func (m *Manager) updateTopology(t *topology.Topology) {
	machines, err := t.ListMachines("")
	if err != nil {
		m.logger.Println(err)
		return
	}
	deleteList := m.updateTopologyLocked(t, machines)
	for _, hypervisor := range deleteList {
		m.ipStorer.UnregisterHypervisor(hypervisor.machine.HostIpAddress)
		hypervisor.delete()
	}
}

func (m *Manager) updateTopologyLocked(t *topology.Topology,
	machines []*topology.Machine) []*hypervisorType {
	hypervisorsToDelete := make(map[string]struct{}, len(machines))
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.topology = t
	for hypervisorName := range m.hypervisors {
		hypervisorsToDelete[hypervisorName] = struct{}{}
	}
	for _, machine := range machines {
		delete(hypervisorsToDelete, machine.Hostname)
		if hypervisor, ok := m.hypervisors[machine.Hostname]; ok {
			go hypervisor.update(machine)
		} else {
			hypervisor := &hypervisorType{
				logger:  prefixlogger.New(machine.Hostname+": ", m.logger),
				machine: machine,
				vms:     make(map[string]*vmInfoType),
			}
			m.hypervisors[machine.Hostname] = hypervisor
			go m.manageHypervisorLoop(hypervisor, machine.Hostname)
		}
	}
	deleteList := make([]*hypervisorType, 0, len(hypervisorsToDelete))
	for hypervisorName := range hypervisorsToDelete {
		deleteList = append(deleteList, m.hypervisors[hypervisorName])
		delete(m.hypervisors, hypervisorName)
	}
	subnetsToDelete := make(map[string]struct{}, len(m.subnets))
	for gatewayIp := range m.subnets {
		subnetsToDelete[gatewayIp] = struct{}{}
	}
	t.Walk(func(directory *topology.Directory) error {
		for _, tSubnet := range directory.Subnets {
			gatewayIp := tSubnet.IpGateway.String()
			delete(subnetsToDelete, gatewayIp)
			m.subnets[gatewayIp] = m.makeSubnet(tSubnet)
		}
		return nil
	})
	for gatewayIp := range subnetsToDelete {
		delete(m.subnets, gatewayIp)
	}
	return deleteList
}

func (h *hypervisorType) update(machine *topology.Machine) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.machine = machine
}

func (h *hypervisorType) delete() {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.deleteScheduled = true
	if h.conn != nil {
		h.conn.Close()
		h.conn = nil
	}
}

func (h *hypervisorType) isDeleteScheduled() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.deleteScheduled
}

func (m *Manager) manageHypervisorLoop(h *hypervisorType, hostname string) {
	for !h.isDeleteScheduled() {
		m.manageHypervisor(h, hostname)
		time.Sleep(time.Second)
	}
}

func (m *Manager) manageHypervisor(h *hypervisorType, hostname string) {
	defer func() {
		h.mutex.Lock()
		defer h.mutex.Unlock()
		h.probeStatus = probeStatusBad
		if h.conn != nil {
			h.conn.Close()
			h.conn = nil
		}
	}()
	client, err := srpc.DialHTTP("tcp",
		fmt.Sprintf("%s:%d", hostname, constants.HypervisorPortNumber),
		time.Minute)
	if err != nil {
		h.logger.Debugln(1, err)
		return
	}
	defer client.Close()
	conn, err := client.Call("Hypervisor.GetUpdates")
	if err != nil {
		h.logger.Println(err)
		return
	}
	h.mutex.Lock()
	h.probeStatus = probeStatusGood
	if h.deleteScheduled {
		h.mutex.Unlock()
		conn.Close()
		return
	}
	h.conn = conn
	h.mutex.Unlock()
	decoder := gob.NewDecoder(conn)
	h.logger.Debugln(0, "waiting for Update messages")
	firstUpdate := true
	for {
		var update proto.Update
		if err := decoder.Decode(&update); err != nil {
			if err != io.EOF {
				h.logger.Println(err)
			}
			return
		}
		m.processHypervisorUpdate(h, update, firstUpdate)
		firstUpdate = false
	}
}

func (m *Manager) processAddressPoolUpdates(h *hypervisorType,
	update proto.Update) {
	if update.HaveAddressPool {
		addresses := make([]net.IP, 0, len(update.AddressPool))
		for _, address := range update.AddressPool {
			addresses = append(addresses, address.IpAddress)
		}
		err := m.ipStorer.SetIPsForHypervisor(h.machine.HostIpAddress,
			addresses)
		if err != nil {
			h.logger.Println(err)
		}
	}
	if update.HaveNumFree {
		if update.NumFreeAddresses < *minimumAddressPoolSize {
			hypervisorAddress := fmt.Sprintf("%s:%d",
				h.machine.Hostname, constants.HypervisorPortNumber)
			m.mutex.Lock()
			defer m.mutex.Unlock()
			tSubnets, err := m.topology.GetSubnetsForMachine(h.machine.Hostname)
			if err != nil {
				h.logger.Println(err)
				return
			}
			freeIPs, err := m.findFreeIPs(tSubnets,
				*desiredAddressPoolSize-update.NumFreeAddresses)
			if err != nil {
				h.logger.Println(err)
				return
			}
			addresses := make([]proto.Address, 0, len(freeIPs))
			for _, ip := range freeIPs {
				addresses = append(addresses, proto.Address{
					IpAddress: ip,
					MacAddress: fmt.Sprintf("52:54:%02x:%02x:%02x:%02x",
						ip[0], ip[1], ip[2], ip[3]),
				})
			}
			client, err := srpc.DialHTTP("tcp", hypervisorAddress, time.Minute)
			if err != nil {
				h.logger.Println(err)
				return
			}
			defer client.Close()
			request := proto.AddAddressesToPoolRequest{addresses}
			var reply proto.AddAddressesToPoolResponse
			err = client.RequestReply("Hypervisor.AddAddressesToPool",
				request, &reply)
			if err == nil {
				err = errors.New(reply.Error)
			}
			if err != nil {
				h.logger.Println(err)
				return
			}
			m.ipStorer.AddIPsForHypervisor(h.machine.HostIpAddress, freeIPs)
			h.logger.Debugf(0, "replenished pool with %d addresses\n",
				len(addresses))
		} else if update.NumFreeAddresses > *maximumAddressPoolSize {
			hypervisorAddress := fmt.Sprintf("%s:%d",
				h.machine.Hostname, constants.HypervisorPortNumber)
			client, err := srpc.DialHTTP("tcp", hypervisorAddress, time.Minute)
			if err != nil {
				h.logger.Println(err)
				return
			}
			defer client.Close()
			request := proto.RemoveExcessAddressesFromPoolRequest{
				*desiredAddressPoolSize}
			var reply proto.RemoveExcessAddressesFromPoolResponse
			err = client.RequestReply(
				"Hypervisor.RemoveExcessAddressesFromPool",
				request, &reply)
			if err == nil {
				err = errors.New(reply.Error)
			}
			if err != nil {
				h.logger.Println(err)
				return
			}
			h.logger.Debugf(0, "removed %d excess addresses from pool\n",
				update.NumFreeAddresses-*desiredAddressPoolSize)
		}
	}
}

func (m *Manager) processHypervisorUpdate(h *hypervisorType,
	update proto.Update, firstUpdate bool) {
	if update.HaveSubnets { // Must do subnets first.
		m.processSubnetsUpdates(h, update.Subnets)
	}
	m.processAddressPoolUpdates(h, update)
	if update.HaveVMs {
		if firstUpdate {
			m.processInitialVMs(h, update.VMs)
		} else {
			m.processVmUpdates(h, update.VMs)
		}
	}
}

func (m *Manager) processInitialVMs(h *hypervisorType,
	vms map[string]*proto.VmInfo) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	vmsToDelete := make(map[string]struct{}, len(h.vms))
	for ipAddr := range h.vms {
		vmsToDelete[ipAddr] = struct{}{}
	}
	for ipAddr, protoVm := range vms {
		delete(vmsToDelete, ipAddr)
		if vm, ok := h.vms[ipAddr]; ok {
			vm.VmInfo = *protoVm
		} else {
			vm := &vmInfoType{ipAddr, *protoVm, h}
			h.vms[ipAddr] = vm
			m.vms[ipAddr] = vm
		}
	}
	for ipAddr := range vmsToDelete {
		delete(h.vms, ipAddr)
		delete(m.vms, ipAddr)
	}
}

func (m *Manager) processSubnetsUpdates(h *hypervisorType,
	haveSubnets []proto.Subnet) {
	haveSubnetsMap := make(map[string]int, len(haveSubnets))
	for index, subnet := range haveSubnets {
		haveSubnetsMap[subnet.Id] = index
	}
	t, err := m.getTopology()
	if err != nil {
		h.logger.Println(err)
		return
	}
	needSubnets, err := t.GetSubnetsForMachine(h.machine.Hostname)
	if err != nil {
		h.logger.Println(err)
		return
	}
	subnetsToAdd := make([]proto.Subnet, 0)
	for _, needSubnet := range needSubnets {
		if index, ok := haveSubnetsMap[needSubnet.Id]; ok {
			haveSubnet := haveSubnets[index]
			if !needSubnet.IpGateway.Equal(haveSubnet.IpGateway) {
				h.logger.Printf("subnet mismatch: %s: %s!=%s\n",
					needSubnet.Id, needSubnet.IpGateway, haveSubnet.IpGateway)
			}
		} else {
			subnetsToAdd = append(subnetsToAdd, needSubnet.Subnet)
		}
	}
	if len(subnetsToAdd) < 1 {
		return
	}
	client, err := srpc.DialHTTP("tcp",
		fmt.Sprintf("%s:%d",
			h.machine.Hostname, constants.HypervisorPortNumber),
		time.Minute)
	if err != nil {
		h.logger.Println(err)
		return
	}
	defer client.Close()
	request := proto.AddSubnetsRequest{Subnets: subnetsToAdd}
	var reply proto.AddSubnetsResponse
	err = client.RequestReply("Hypervisor.AddSubnets", request, &reply)
	if err == nil {
		err = errors.New(reply.Error)
	}
	if err != nil {
		h.logger.Println(err)
		return
	}
	h.logger.Debugf(0, "Added %d subnets\n", len(subnetsToAdd))
}

func (m *Manager) processVmUpdates(h *hypervisorType,
	updateVMs map[string]*proto.VmInfo) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for ipAddr, protoVm := range updateVMs {
		if len(protoVm.Volumes) < 1 {
			delete(h.vms, ipAddr)
			delete(m.vms, ipAddr)
		} else {
			if vm, ok := h.vms[ipAddr]; ok {
				vm.VmInfo = *protoVm
			} else {
				vm := &vmInfoType{ipAddr, *protoVm, h}
				h.vms[ipAddr] = vm
				m.vms[ipAddr] = vm
			}
		}
	}
}
