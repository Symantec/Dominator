package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/mdb"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

var emptyTags = make(map[string]string)

type hypervisorGeneratorType struct {
	logger       log.DebugLogger
	eventChannel chan<- struct{}
	mutex        sync.Mutex
	vms          map[string]*proto.VmInfo
}

func newHypervisorGenerator(args []string,
	logger log.DebugLogger) (generator, error) {
	g := &hypervisorGeneratorType{
		logger: logger,
		vms:    make(map[string]*proto.VmInfo),
	}
	go g.daemon()
	return g, nil
}

func (g *hypervisorGeneratorType) daemon() {
	address := fmt.Sprintf(":%d", constants.HypervisorPortNumber)
	for {
		if err := g.getUpdates(address); err != nil {
			g.logger.Println(err)
			time.Sleep(time.Second)
		}
	}
}

func (g *hypervisorGeneratorType) getUpdates(hypervisor string) error {
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	conn, err := client.Call("Hypervisor.GetUpdates")
	if err != nil {
		return err
	}
	defer conn.Close()
	initialUpdate := true
	for {
		var update proto.Update
		if err := conn.Decode(&update); err != nil {
			return err
		}
		g.updateVMs(update.VMs, initialUpdate)
		initialUpdate = false
		select {
		case g.eventChannel <- struct{}{}:
		default:
		}
	}
}

func (g *hypervisorGeneratorType) Generate(unused_datacentre string,
	logger log.Logger) (*mdb.Mdb, error) {
	var newMdb mdb.Mdb
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for ipAddr, vm := range g.vms {
		if vm.State == proto.StateRunning {
			tags := vm.Tags
			if tags == nil {
				tags = emptyTags
			}
			_, disableUpdates := tags["DisableUpdates"]
			var ownerGroup string
			if len(vm.OwnerGroups) > 0 {
				ownerGroup = vm.OwnerGroups[0]
			}
			newMdb.Machines = append(newMdb.Machines, mdb.Machine{
				Hostname:       ipAddr,
				IpAddress:      ipAddr,
				RequiredImage:  tags["RequiredImage"],
				PlannedImage:   tags["PlannedImage"],
				DisableUpdates: disableUpdates,
				OwnerGroup:     ownerGroup,
				Tags:           vm.Tags,
			})
		}
	}
	return &newMdb, nil
}

func (g *hypervisorGeneratorType) RegisterEventChannel(events chan<- struct{}) {
	g.eventChannel = events
}

func (g *hypervisorGeneratorType) updateVMs(vms map[string]*proto.VmInfo,
	initialUpdate bool) {
	vmsToDelete := make(map[string]struct{}, len(g.vms))
	if initialUpdate {
		for ipAddr := range g.vms {
			vmsToDelete[ipAddr] = struct{}{}
		}
	}
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for ipAddr, vm := range vms {
		if vm == nil || len(vm.Volumes) < 1 {
			delete(g.vms, ipAddr)
		} else {
			g.vms[ipAddr] = vm
			delete(vmsToDelete, ipAddr)
		}
	}
	for ipAddr := range vmsToDelete {
		delete(g.vms, ipAddr)
	}
}
