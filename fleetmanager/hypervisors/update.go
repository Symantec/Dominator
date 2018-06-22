package hypervisors

import (
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
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
	manageHypervisors = flag.Bool("manageHypervisors", false,
		"If true, manage hypervisors")
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

func (m *Manager) updateHypervisor(h *hypervisorType,
	machine *topology.Machine) {
	h.mutex.Lock()
	h.machine = machine
	subnets := h.subnets
	h.mutex.Unlock()
	if *manageHypervisors && h.probeStatus == probeStatusGood {
		m.processSubnetsUpdates(h, subnets)
	}
}

func (m *Manager) updateTopology(t *topology.Topology) {
	machines, err := t.ListMachines("")
	if err != nil {
		m.logger.Println(err)
		return
	}
	deleteList := m.updateTopologyLocked(t, machines)
	for _, hypervisor := range deleteList {
		m.storer.UnregisterHypervisor(hypervisor.machine.HostIpAddress)
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
			go m.updateHypervisor(hypervisor, machine)
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
	vmList, err := m.storer.ListVMs(h.machine.HostIpAddress)
	if err != nil {
		h.logger.Printf("error reading VMs, not managing hypervisor: %s", err)
		return
	}
	for _, vmIpAddr := range vmList {
		pVmInfo, err := m.storer.ReadVm(h.machine.HostIpAddress, vmIpAddr)
		if err != nil {
			h.logger.Printf("error reading VM: %s, not managing hypervisor: %s",
				vmIpAddr, err)
		}
		vmInfo := &vmInfoType{vmIpAddr, *pVmInfo, h}
		h.vms[vmIpAddr] = vmInfo
		m.mutex.Lock()
		m.vms[vmIpAddr] = vmInfo
		m.mutex.Unlock()
	}
	for !h.isDeleteScheduled() {
		sleepTime := m.manageHypervisor(h, hostname)
		time.Sleep(sleepTime)
	}
}

func (m *Manager) manageHypervisor(h *hypervisorType,
	hostname string) time.Duration {
	failureProbeStatus := probeStatusBad
	defer func() {
		h.mutex.Lock()
		defer h.mutex.Unlock()
		h.probeStatus = failureProbeStatus
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
		if err == srpc.ErrorNoSrpcEndpoint {
			failureProbeStatus = probeStatusNoSrpc
		}
		return time.Second
	}
	defer client.Close()
	conn, err := client.Call("Hypervisor.GetUpdates")
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown service") {
			h.logger.Debugln(1, err)
			failureProbeStatus = probeStatusNoService
			return time.Minute
		} else {
			h.logger.Println(err)
		}
		return time.Second
	}
	h.mutex.Lock()
	h.probeStatus = probeStatusGood
	if h.deleteScheduled {
		h.mutex.Unlock()
		conn.Close()
		return 0
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
			return time.Second
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
		err := m.storer.SetIPsForHypervisor(h.machine.HostIpAddress,
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
			m.storer.AddIPsForHypervisor(h.machine.HostIpAddress, freeIPs)
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
	if *manageHypervisors {
		if update.HaveSubnets { // Must do subnets first.
			h.mutex.Lock()
			h.subnets = update.Subnets
			h.mutex.Unlock()
			m.processSubnetsUpdates(h, update.Subnets)
		}
		m.processAddressPoolUpdates(h, update)
	}
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
		if _, ok := vms[ipAddr]; !ok {
			vmsToDelete[ipAddr] = struct{}{}
		}
	}
	m.processVmUpdatesWithLock(h, vms, vmsToDelete)
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
	subnetsToDelete := make(map[string]struct{}, len(haveSubnets))
	for _, subnet := range haveSubnets {
		subnetsToDelete[subnet.Id] = struct{}{}
	}
	var request proto.UpdateSubnetsRequest
	for _, needSubnet := range needSubnets {
		if index, ok := haveSubnetsMap[needSubnet.Id]; ok {
			haveSubnet := haveSubnets[index]
			delete(subnetsToDelete, haveSubnet.Id)
			if !needSubnet.Equal(&haveSubnet) {
				request.Change = append(request.Change, needSubnet.Subnet)
			}
		} else {
			request.Add = append(request.Add, needSubnet.Subnet)
		}
	}
	for subnetId := range subnetsToDelete {
		request.Delete = append(request.Delete, subnetId)
	}
	if len(request.Add) < 1 && len(request.Change) < 1 &&
		len(request.Delete) < 1 {
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
	var reply proto.UpdateSubnetsResponse
	err = client.RequestReply("Hypervisor.UpdateSubnets", request, &reply)
	if err == nil {
		err = errors.New(reply.Error)
	}
	if err != nil {
		h.logger.Println(err)
		return
	}
	h.logger.Debugf(0, "Added %d, changed %d and deleted %d subnets\n",
		len(request.Add), len(request.Change), len(request.Delete))
}

func (m *Manager) processVmUpdates(h *hypervisorType,
	updateVMs map[string]*proto.VmInfo) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.processVmUpdatesWithLock(h, updateVMs, make(map[string]struct{}))
}

func (m *Manager) processVmUpdatesWithLock(h *hypervisorType,
	updateVMs map[string]*proto.VmInfo, vmsToDelete map[string]struct{}) {
	for ipAddr, protoVm := range updateVMs {
		if len(protoVm.Volumes) < 1 {
			vmsToDelete[ipAddr] = struct{}{}
		} else {
			if vm, ok := h.vms[ipAddr]; ok {
				if !vm.VmInfo.Equal(protoVm) {
					err := m.storer.WriteVm(h.machine.HostIpAddress, ipAddr,
						*protoVm)
					if err != nil {
						h.logger.Printf("error writing VM: %s: %s\n",
							ipAddr, err)
					} else {
						h.logger.Debugf(0, "updated VM: %s\n", ipAddr)
					}
				}
				vm.VmInfo = *protoVm
			} else {
				vm := &vmInfoType{ipAddr, *protoVm, h}
				h.vms[ipAddr] = vm
				m.vms[ipAddr] = vm
				err := m.storer.WriteVm(h.machine.HostIpAddress, ipAddr,
					*protoVm)
				if err != nil {
					h.logger.Printf("error writing VM: %s: %s\n", ipAddr, err)
				} else {
					h.logger.Debugf(0, "wrote VM: %s\n", ipAddr)
				}
			}
		}
	}
	for ipAddr := range vmsToDelete {
		delete(h.vms, ipAddr)
		delete(m.vms, ipAddr)
		err := m.storer.DeleteVm(h.machine.HostIpAddress, ipAddr)
		if err != nil {
			h.logger.Printf("error deleting VM: %s: %s\n", ipAddr, err)
		} else {
			h.logger.Debugf(0, "deleted VM: %s\n", ipAddr)
		}
	}
}
