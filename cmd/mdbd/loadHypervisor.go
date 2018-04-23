package main

import (
	"encoding/gob"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
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
	for {
		if err := g.getUpdates("localhost:6976"); err != nil {
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
	decoder := gob.NewDecoder(conn)
	for {
		var update proto.Update
		if err := decoder.Decode(&update); err != nil {
			return err
		}
		g.updateVMs(update.VMs)
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

func (g *hypervisorGeneratorType) updateVMs(vms map[string]*proto.VmInfo) {
	if len(vms) < 1 {
		return
	}
	g.mutex.Lock()
	defer g.mutex.Unlock()
	for ipAddr, vm := range vms {
		if vm == nil {
			delete(g.vms, ipAddr)
		} else {
			g.vms[ipAddr] = vm
		}
	}
}
