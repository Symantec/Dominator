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
	"github.com/Symantec/Dominator/lib/tags"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
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

func testInLocation(location, enclosingLocation string) bool {
	if enclosingLocation != "" && location != enclosingLocation {
		if len(enclosingLocation) >= len(location) {
			return false
		}
		if location[len(enclosingLocation)] != '/' {
			return false
		}
		if location[:len(enclosingLocation)] != enclosingLocation {
			return false
		}
	}
	return true
}

func (m *Manager) changeMachineTags(hostname string, tgs tags.Tags) error {
	if !*manageHypervisors {
		return errors.New("this is a read-only Fleet Manager")
	}
	if h, err := m.getLockedHypervisor(hostname, true); err != nil {
		return err
	} else {
		for key, localVal := range tgs { // Delete duplicates.
			if machineVal := h.machine.Tags[key]; localVal == machineVal {
				delete(tgs, key)
			}
		}
		err := m.storer.WriteMachineTags(h.machine.HostIpAddress, tgs)
		if err != nil {
			return err
		}
		if len(tgs) > 0 {
			h.localTags = tgs
		} else {
			h.localTags = nil
		}
		update := &fm_proto.Update{
			ChangedMachines: []*fm_proto.Machine{h.getMachineLocked()},
		}
		location := h.location
		h.mutex.Unlock()
		m.sendUpdate(location, update)
		return nil
	}
}

func (h *hypervisorType) getMachine() *fm_proto.Machine {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.getMachineLocked()
}

func (h *hypervisorType) getMachineLocked() *fm_proto.Machine {
	if len(h.localTags) < 1 {
		return h.machine
	}
	var machine fm_proto.Machine
	machine = *h.machine
	machine.Tags = h.machine.Tags.Copy()
	machine.Tags.Merge(h.localTags)
	return &machine
}

func (m *Manager) closeUpdateChannel(channel <-chan fm_proto.Update) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.notifiers[channel].notifiers, channel)
	delete(m.notifiers, channel)
}

func (m *Manager) makeUpdateChannel(locationStr string) <-chan fm_proto.Update {
	channel := make(chan fm_proto.Update, 16)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.locations == nil {
		m.locations = make(map[string]*locationType)
	}
	if m.notifiers == nil {
		m.notifiers = make(map[<-chan fm_proto.Update]*locationType)
	}
	location, ok := m.locations[locationStr]
	if !ok {
		location = &locationType{
			notifiers: make(map[<-chan fm_proto.Update]chan<- fm_proto.Update),
		}
		m.locations[locationStr] = location
	}
	location.notifiers[channel] = channel
	m.notifiers[channel] = location
	if !*manageHypervisors {
		channel <- fm_proto.Update{Error: "this is a read-only Fleet Manager"}
		return channel
	}
	machines := make([]*fm_proto.Machine, 0)
	vms := make(map[string]*hyper_proto.VmInfo, len(m.vms))
	for _, h := range m.hypervisors {
		if !testInLocation(h.location, locationStr) {
			continue
		}
		machines = append(machines, h.getMachine())
		for addr, vm := range h.vms {
			vms[addr] = &vm.VmInfo
		}
	}
	channel <- fm_proto.Update{
		ChangedMachines: machines,
		ChangedVMs:      vms,
	}
	return channel
}

func (m *Manager) updateHypervisor(h *hypervisorType,
	machine *fm_proto.Machine) {
	location, _ := m.topology.GetLocationOfMachine(machine.Hostname)
	var numTagsToDelete uint
	h.mutex.Lock()
	h.location = location
	h.machine = machine
	subnets := h.subnets
	for key, localVal := range h.localTags {
		if machineVal, ok := h.machine.Tags[key]; ok && localVal == machineVal {
			delete(h.localTags, key)
			numTagsToDelete++
		}
	}
	if numTagsToDelete > 0 {
		err := m.storer.WriteMachineTags(h.machine.HostIpAddress, h.localTags)
		if err != nil {
			h.logger.Printf("error writing tags: %s\n", err)
		} else {
			h.logger.Debugf(0, "Deleted %d obsolete local tags\n",
				numTagsToDelete)
		}
	}
	h.mutex.Unlock()
	if *manageHypervisors && h.probeStatus == probeStatusGood {
		go m.processSubnetsUpdates(h, subnets)
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
	machines []*fm_proto.Machine) []*hypervisorType {
	hypervisorsToDelete := make(map[string]struct{}, len(machines))
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.topology = t
	for hypervisorName := range m.hypervisors {
		hypervisorsToDelete[hypervisorName] = struct{}{}
	}
	var hypersToChange, hypersToDelete []*hypervisorType
	for _, machine := range machines {
		delete(hypervisorsToDelete, machine.Hostname)
		if hypervisor, ok := m.hypervisors[machine.Hostname]; ok {
			if !hypervisor.machine.Equal(machine) {
				hypersToChange = append(hypersToChange, hypervisor)
			}
			m.updateHypervisor(hypervisor, machine)
		} else {
			location, _ := m.topology.GetLocationOfMachine(machine.Hostname)
			hypervisor := &hypervisorType{
				logger:   prefixlogger.New(machine.Hostname+": ", m.logger),
				location: location,
				machine:  machine,
				vms:      make(map[string]*vmInfoType),
			}
			m.hypervisors[machine.Hostname] = hypervisor
			hypersToChange = append(hypersToChange, hypervisor)
			go m.manageHypervisorLoop(hypervisor, machine.Hostname)
		}
	}
	deleteList := make([]*hypervisorType, 0, len(hypervisorsToDelete))
	for hypervisorName := range hypervisorsToDelete {
		hypervisor := m.hypervisors[hypervisorName]
		deleteList = append(deleteList, hypervisor)
		delete(m.hypervisors, hypervisorName)
		hypersToDelete = append(hypersToDelete, hypervisor)
	}
	if len(hypersToChange) > 0 || len(hypersToDelete) > 0 {
		updates := m.splitChanges(hypersToChange, hypersToDelete)
		for location, updateForLocation := range updates {
			m.sendUpdate(location, updateForLocation)
		}
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
	h.localTags, err = m.storer.ReadMachineTags(h.machine.HostIpAddress)
	if err != nil {
		h.logger.Printf("error reading tags, not managing hypervisor: %s", err)
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
		var update hyper_proto.Update
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
	update hyper_proto.Update) {
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
			if len(freeIPs) < 1 {
				return
			}
			addresses := make([]hyper_proto.Address, 0, len(freeIPs))
			for _, ip := range freeIPs {
				addresses = append(addresses, hyper_proto.Address{
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
			request := hyper_proto.AddAddressesToPoolRequest{addresses}
			var reply hyper_proto.AddAddressesToPoolResponse
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
			request := hyper_proto.RemoveExcessAddressesFromPoolRequest{
				*desiredAddressPoolSize}
			var reply hyper_proto.RemoveExcessAddressesFromPoolResponse
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
	update hyper_proto.Update, firstUpdate bool) {
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
	vms map[string]*hyper_proto.VmInfo) {
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
	haveSubnets []hyper_proto.Subnet) {
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
	var request hyper_proto.UpdateSubnetsRequest
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
	var reply hyper_proto.UpdateSubnetsResponse
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
	updateVMs map[string]*hyper_proto.VmInfo) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.processVmUpdatesWithLock(h, updateVMs, make(map[string]struct{}))
}

func (m *Manager) processVmUpdatesWithLock(h *hypervisorType,
	updateVMs map[string]*hyper_proto.VmInfo, vmsToDelete map[string]struct{}) {
	update := fm_proto.Update{ChangedVMs: make(map[string]*hyper_proto.VmInfo)}
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
			update.ChangedVMs[ipAddr] = protoVm
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
		update.DeletedVMs = append(update.DeletedVMs, ipAddr)
	}
	m.sendUpdate(h.location, &update)
}

func (m *Manager) splitChanges(hypersToChange []*hypervisorType,
	hypersToDelete []*hypervisorType) map[string]*fm_proto.Update {
	updates := make(map[string]*fm_proto.Update)
	for _, h := range hypersToChange {
		if locationUpdate, ok := updates[h.location]; !ok {
			updates[h.location] = &fm_proto.Update{
				ChangedMachines: []*fm_proto.Machine{h.getMachine()},
			}
		} else {
			locationUpdate.ChangedMachines = append(
				locationUpdate.ChangedMachines, h.getMachine())
		}
	}
	for _, h := range hypersToDelete {
		if locationUpdate, ok := updates[h.location]; !ok {
			updates[h.location] = &fm_proto.Update{
				DeletedMachines: []string{h.machine.Hostname},
			}
		} else {
			locationUpdate.DeletedMachines = append(
				locationUpdate.DeletedMachines, h.machine.Hostname)
		}
	}
	return updates
}

func (m *Manager) sendUpdate(hyperLocation string, update *fm_proto.Update) {
	if len(update.ChangedMachines) < 1 && len(update.ChangedVMs) < 1 &&
		len(update.DeletedMachines) < 1 && len(update.DeletedVMs) < 1 {
		return
	}
	for locationStr, location := range m.locations {
		if !testInLocation(hyperLocation, locationStr) {
			continue
		}
		for _, channel := range location.notifiers {
			channel <- *update
		}
	}
}
