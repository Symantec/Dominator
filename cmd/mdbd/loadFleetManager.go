package main

import (
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/srpc"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type fleetManagerGeneratorType struct {
	fleetManager string
	location     string
	logger       log.DebugLogger
	eventChannel chan<- struct{}
	mutex        sync.Mutex
	machines     map[string]*fm_proto.Machine
	vms          map[string]*hyper_proto.VmInfo
}

func newFleetManagerGenerator(args []string,
	logger log.DebugLogger) (generator, error) {
	g := &fleetManagerGeneratorType{
		fleetManager: fmt.Sprintf("%s:%d",
			args[0], constants.FleetManagerPortNumber),
		logger:   logger,
		machines: make(map[string]*fm_proto.Machine),
		vms:      make(map[string]*hyper_proto.VmInfo),
	}
	if len(args) > 1 {
		g.location = args[1]
	}
	go g.daemon()
	return g, nil
}

func (g *fleetManagerGeneratorType) daemon() {
	for {
		if err := g.getUpdates(g.fleetManager); err != nil {
			g.logger.Println(err)
			time.Sleep(time.Second)
		}
	}
}

func (g *fleetManagerGeneratorType) getUpdates(fleetManager string) error {
	client, err := srpc.DialHTTP("tcp", g.fleetManager, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	conn, err := client.Call("FleetManager.GetUpdates")
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	request := fm_proto.GetUpdatesRequest{Location: g.location}
	if err := encoder.Encode(request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	decoder := gob.NewDecoder(conn)
	for {
		var update fm_proto.Update
		if err := decoder.Decode(&update); err != nil {
			return err
		}
		g.update(update)
		select {
		case g.eventChannel <- struct{}{}:
		default:
		}
	}
}

func (g *fleetManagerGeneratorType) Generate(unused_datacentre string,
	logger log.Logger) (*mdb.Mdb, error) {
	var newMdb mdb.Mdb
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for _, machine := range g.machines {
		var ipAddr string
		if len(machine.HostIpAddress) > 0 {
			ipAddr = machine.HostIpAddress.String()
		}
		newMdb.Machines = append(newMdb.Machines, mdb.Machine{
			Hostname:  machine.Hostname,
			IpAddress: ipAddr,
		})
	}
	for ipAddr, vm := range g.vms {
		if vm.State == hyper_proto.StateRunning {
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

func (g *fleetManagerGeneratorType) RegisterEventChannel(
	events chan<- struct{}) {
	g.eventChannel = events
}

func (g *fleetManagerGeneratorType) update(update fm_proto.Update) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for _, machine := range update.ChangedMachines {
		g.machines[machine.Hostname] = machine
	}
	for _, hostname := range update.DeletedMachines {
		delete(g.machines, hostname)
	}
	for ipAddr, vm := range update.ChangedVMs {
		g.vms[ipAddr] = vm
	}
	for _, ipAddr := range update.DeletedVMs {
		delete(g.vms, ipAddr)
	}
}
