package main

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	imageclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/concurrent"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/cpusharer"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	libnet "github.com/Symantec/Dominator/lib/net"
	"github.com/Symantec/Dominator/lib/rpcclientpool"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/tags"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
	sub_proto "github.com/Symantec/Dominator/proto/sub"
	subclient "github.com/Symantec/Dominator/sub/client"
	"github.com/Symantec/tricorder/go/tricorder/messages"
)

const (
	filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP |
		syscall.S_IROTH
)

type hypervisorType struct {
	healthAgentClientResource *rpcclientpool.ClientResource
	hostname                  string
	hypervisorClientResource  *srpc.ClientResource
	initialTags               tags.Tags
	initialUnhealthyList      map[string]struct{}
	logger                    log.DebugLogger
	noVMs                     bool
	subClientResource         *srpc.ClientResource
}

func rolloutImageSubcommand(args []string, logger log.DebugLogger) {
	err := rolloutImage(args[0], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rolling out image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func gitCommand(repositoryDirectory string, command ...string) ([]byte, error) {
	cmd := exec.Command("git", command...)
	cmd.Dir = repositoryDirectory
	cmd.Stderr = os.Stderr
	if output, err := cmd.Output(); err != nil {
		return nil, err
	} else {
		return output, nil
	}
}

func rolloutImage(imageName string, logger log.DebugLogger) error {
	startTime := time.Now()
	cpuSharer := cpusharer.NewFifoCpuSharer()
	if *topologyDir != "" {
		logger.Debugln(0, "updating Git repository")
		stdout, err := gitCommand(*topologyDir, "status", "--porcelain")
		if err != nil {
			return err
		}
		if len(stdout) > 0 {
			return errors.New("Git repository is not clean")
		}
		if _, err := gitCommand(*topologyDir, "pull"); err != nil {
			return err
		}
	}
	logger.Debugln(0, "checking image")
	if foundImage, err := checkImage(imageName); err != nil {
		return err
	} else if !foundImage {
		return fmt.Errorf("image: %s not found", imageName)
	}
	fleetManagerClientResource := srpc.NewClientResource("tcp",
		fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum))
	defer fleetManagerClientResource.ScheduleClose()
	logger.Debugln(0, "finding good Hypervisors")
	hypervisorAddresses, err := listGoodHypervisors(fleetManagerClientResource)
	if err != nil {
		return err
	}
	hypervisors := make([]*hypervisorType, 0, len(hypervisorAddresses))
	defer closeHypervisors(hypervisors)
	tagsForHypervisors, err := getTagsForHypervisors(fleetManagerClientResource)
	logger.Debugln(0, "checking and tagging Hypervisors")
	if err != nil {
		return fmt.Errorf("failure getting tags: %s", err)
	}
	hypervisorsChannel := make(chan *hypervisorType, len(hypervisorAddresses))
	for _, address := range hypervisorAddresses {
		if hostname, _, err := net.SplitHostPort(address); err != nil {
			return err
		} else {
			go func(hostname string) {
				cpuSharer.GrabCpu()
				defer cpuSharer.ReleaseCpu()
				hypervisor := setupHypervisor(hostname, imageName,
					tagsForHypervisors[hostname], cpuSharer, logger)
				hypervisorsChannel <- hypervisor
			}(hostname)
		}
	}
	for range hypervisorAddresses {
		if hypervisor := <-hypervisorsChannel; hypervisor != nil {
			err := hypervisor.updateTagForHypervisor(
				fleetManagerClientResource, "PlannedImage", imageName)
			if err != nil {
				return fmt.Errorf("%s: failure updating tags: %s",
					hypervisor.hostname, err)
			}
			hypervisors = append(hypervisors, hypervisor)
		}
	}
	if len(hypervisors) < 1 {
		return errors.New("no hypervisors to update")
	}
	logger.Debugln(0, "splitting unused/used Hypervisors")
	unusedHypervisors, usedHypervisors := markUnusedHypervisors(hypervisors,
		cpuSharer)
	logger.Debugf(0, "%d unused, %d used Hypervisors\n",
		len(unusedHypervisors), len(usedHypervisors))
	logger.Debugln(0, "upgrading unused Hypervisors")
	err = upgradeOneThenAll(fleetManagerClientResource, imageName,
		unusedHypervisors, cpuSharer, uint(len(unusedHypervisors)))
	if err != nil {
		return err
	}
	numConcurrent := uint(len(usedHypervisors) / 2)
	if numConcurrent < 1 {
		numConcurrent = 1
	} else if numConcurrent > uint(len(unusedHypervisors)) {
		numConcurrent = 1
	} else if numConcurrent*10 < uint(len(usedHypervisors)) {
		numConcurrent++
	}
	logger.Debugln(0, "upgrading used Hypervisors")
	err = upgradeOneThenAll(fleetManagerClientResource, imageName,
		usedHypervisors, cpuSharer, numConcurrent)
	if err != nil {
		return err
	}
	if *topologyDir != "" {
		var tgs tags.Tags
		tagsFilename := filepath.Join(*topologyDir, *location, "tags.json")
		if err := json.ReadFromFile(tagsFilename, &tgs); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			tgs = make(tags.Tags)
		}
		oldImageName := tgs["RequiredImage"]
		tgs["RequiredImage"] = imageName
		delete(tgs, "PlannedImage")
		err := json.WriteToFile(tagsFilename, filePerms, "    ", tgs)
		if err != nil {
			return err
		}
		if _, err := gitCommand(*topologyDir, "add", tagsFilename); err != nil {
			return err
		}
		var locationInsert string
		if *location != "" {
			locationInsert = "in " + *location + " "
		}
		_, err = gitCommand(*topologyDir, "commit", "-m",
			fmt.Sprintf("Upgrade %sfrom %s to %s",
				locationInsert, oldImageName, imageName))
		if err != nil {
			return err
		}
		if _, err := gitCommand(*topologyDir, "push"); err != nil {
			return err
		}
	}
	logger.Printf("rollout completed in %s\n",
		format.Duration(time.Since(startTime)))
	return nil
}

func checkImage(imageName string) (bool, error) {
	clientName := fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		return false, err
	}
	defer client.Close()
	return imageclient.CheckImage(client, imageName)
}

func closeHypervisors(hypervisors []*hypervisorType) {
	for _, hypervisor := range hypervisors {
		hypervisor.hypervisorClientResource.ScheduleClose()
		hypervisor.subClientResource.ScheduleClose()
	}
}

func getTagsForHypervisors(clientResource *srpc.ClientResource) (
	map[string]tags.Tags, error) {
	client, err := clientResource.GetHTTP(nil, 0)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	conn, err := client.Call("FleetManager.GetUpdates")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	request := fm_proto.GetUpdatesRequest{Location: *location, MaxUpdates: 1}
	if err := encoder.Encode(request); err != nil {
		return nil, err
	}
	if err := conn.Flush(); err != nil {
		return nil, err
	}
	var reply fm_proto.Update
	if err := decoder.Decode(&reply); err != nil {
		return nil, err
	}
	if err := errors.New(reply.Error); err != nil {
		return nil, err
	}
	tagsForHypervisors := make(map[string]tags.Tags, len(reply.ChangedMachines))
	for _, machine := range reply.ChangedMachines {
		tagsForHypervisors[machine.Hostname] = machine.Tags
	}
	return tagsForHypervisors, nil
}

func listGoodHypervisors(clientResource *srpc.ClientResource) (
	[]string, error) {
	return listGoodHypervisorsInLocation(clientResource, *location)
}

func listGoodHypervisorsInLocation(clientResource *srpc.ClientResource,
	location string) ([]string, error) {
	client, err := clientResource.GetHTTP(nil, 0)
	if err != nil {
		return nil, err
	}
	defer client.Put()
	request := fm_proto.ListHypervisorsInLocationRequest{Location: location}
	var reply fm_proto.ListHypervisorsInLocationResponse
	err = client.RequestReply("FleetManager.ListHypervisorsInLocation",
		request, &reply)
	if err != nil {
		return nil, err
	}
	if err := errors.New(reply.Error); err != nil {
		return nil, err
	}
	return reply.HypervisorAddresses, nil
}

func markUnusedHypervisors(hypervisors []*hypervisorType,
	cpuSharer cpusharer.CpuSharer) (
	map[*hypervisorType]struct{}, map[*hypervisorType]struct{}) {
	dialer := libnet.NewCpuSharingDialer(&net.Dialer{}, cpuSharer)
	waitGroup := &sync.WaitGroup{}
	for _, hypervisor_ := range hypervisors {
		waitGroup.Add(1)
		go func(h *hypervisorType) {
			defer waitGroup.Done()
			cpuSharer.GrabCpu()
			defer cpuSharer.ReleaseCpu()
			client, err := h.hypervisorClientResource.GetHTTPWithDialer(nil,
				dialer)
			if err != nil {
				h.logger.Printf("error connecting to hypervisor: %s\n", err)
				return
			}
			defer client.Put()
			request := hyper_proto.ListVMsRequest{}
			var reply hyper_proto.ListVMsResponse
			err = client.RequestReply("Hypervisor.ListVMs", request, &reply)
			if err != nil {
				h.logger.Printf("error listing VMS: %s", err)
				return
			}
			if len(reply.IpAddresses) < 1 {
				h.noVMs = true
			}
		}(hypervisor_)
	}
	waitGroup.Wait()
	unusedHypervisors := make(map[*hypervisorType]struct{})
	usedHypervisors := make(map[*hypervisorType]struct{})
	for _, hypervisor := range hypervisors {
		if hypervisor.noVMs {
			unusedHypervisors[hypervisor] = struct{}{}
		} else {
			usedHypervisors[hypervisor] = struct{}{}
		}
	}
	return unusedHypervisors, usedHypervisors
}

func setupHypervisor(hostname string, imageName string, tgs tags.Tags,
	cpuSharer *cpusharer.FifoCpuSharer,
	logger log.DebugLogger) *hypervisorType {
	logger = prefixlogger.New(hostname+": ", logger)
	currentRequiredImage := tgs["RequiredImage"]
	if currentRequiredImage != "" &&
		path.Dir(currentRequiredImage) != path.Dir(imageName) {
		logger.Printf(
			"image stream: current=%s != new=%s, skipping\n",
			path.Dir(currentRequiredImage), path.Dir(imageName))
		return nil
	}
	h := &hypervisorType{
		healthAgentClientResource: rpcclientpool.New("tcp",
			fmt.Sprintf("%s:%d", hostname, 6910), true, ""),
		hostname: hostname,
		hypervisorClientResource: srpc.NewClientResource("tcp",
			fmt.Sprintf("%s:%d", hostname,
				constants.HypervisorPortNumber)),
		initialTags:          tgs,
		initialUnhealthyList: make(map[string]struct{}),
		logger:               logger,
		subClientResource: srpc.NewClientResource("tcp",
			fmt.Sprintf("%s:%d", hostname, constants.SubPortNumber)),
	}
	if lastImage, err := h.getLastImageName(cpuSharer); err != nil {
		logger.Printf("skipping: %s\n", err)
		return nil
	} else if lastImage == imageName {
		logger.Println("already updated, skipping")
		return nil
	} else {
		return h
	}
}

func upgradeOneThenAll(fleetManagerClientResource *srpc.ClientResource,
	imageName string, hypervisors map[*hypervisorType]struct{},
	cpuSharer *cpusharer.FifoCpuSharer, maxConcurrent uint) error {
	if len(hypervisors) < 1 {
		return nil
	}
	state := concurrent.NewStateWithLinearConcurrencyIncrease(1, maxConcurrent)
	for hypervisor := range hypervisors {
		hypervisor := hypervisor
		err := state.GoRun(func() error {
			err := hypervisor.upgrade(fleetManagerClientResource, imageName,
				cpuSharer)
			if err != nil {
				return fmt.Errorf("error upgrading: %s: %s",
					hypervisor.hostname, err)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return state.Reap()
}

func (h *hypervisorType) getFailingHealthChecks(
	cpuSharer *cpusharer.FifoCpuSharer,
	timeout time.Duration) ([]string, time.Time, error) {
	stopTime := time.Now().Add(timeout)
	for ; time.Until(stopTime) >= 0; cpuSharer.Sleep(time.Second) {
		if list, timestamp, err := h.getFailingHealthChecksOnce(); err == nil {
			return list, timestamp, nil
		}
	}
	return nil, time.Time{}, errors.New("timed out getting health status")
}

func (h *hypervisorType) getFailingHealthChecksOnce() (
	[]string, time.Time, error) {
	client, err := h.healthAgentClientResource.Get(nil)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer client.Put()
	var metric messages.Metric
	err = client.Call("MetricsServer.GetMetric",
		"/health-checks/*/unhealthy-list", &metric)
	if err != nil {
		client.Close()
		return nil, time.Time{}, err
	}
	if list, ok := metric.Value.([]string); !ok {
		return nil, time.Time{}, errors.New("list metric is not []string")
	} else {
		if timestamp, ok := metric.TimeStamp.(time.Time); ok {
			return list, timestamp, nil
		} else {
			return list, time.Time{}, nil
		}
	}
}

func (h *hypervisorType) getLastImageName(cpuSharer *cpusharer.FifoCpuSharer) (
	string, error) {
	client, err := h.subClientResource.GetHTTP(nil, time.Second*15)
	if err != nil {
		return "", fmt.Errorf("error connecting to sub: %s", err)
	}
	defer client.Put()
	request := sub_proto.PollRequest{ShortPollOnly: true}
	var reply sub_proto.PollResponse
	if err := subclient.CallPoll(client, request, &reply); err != nil {
		client.Close()
		if err != io.EOF {
			return "", fmt.Errorf("error polling sub: %s", err)
		}
	}
	return reply.LastSuccessfulImageName, nil
}

func (h *hypervisorType) updateTagForHypervisor(
	clientResource *srpc.ClientResource, key, value string) error {
	newTags := h.initialTags.Copy()
	newTags[key] = value
	if key == "RequiredImage" {
		delete(newTags, "PlannedImage")
	}
	if h.initialTags.Equal(newTags) {
		return nil
	}
	client, err := clientResource.GetHTTP(nil, 0)
	if err != nil {
		return err
	}
	defer client.Put()
	request := fm_proto.ChangeMachineTagsRequest{
		Hostname: h.hostname,
		Tags:     newTags,
	}
	var reply fm_proto.ChangeMachineTagsResponse
	err = client.RequestReply("FleetManager.ChangeMachineTags",
		request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}

func (h *hypervisorType) upgrade(clientResource *srpc.ClientResource,
	imageName string, cpuSharer *cpusharer.FifoCpuSharer) error {
	cpuSharer.GrabCpu()
	defer cpuSharer.ReleaseCpu()
	list, _, err := h.getFailingHealthChecks(cpuSharer, time.Second)
	if err != nil {
		h.logger.Println(err)
		return nil
	} else if len(list) > 0 {
		for _, failed := range list {
			h.initialUnhealthyList[failed] = struct{}{}
		}
	}
	h.logger.Debugln(0, "upgrading")
	err = h.updateTagForHypervisor(clientResource, "RequiredImage", imageName)
	if err != nil {
		return err
	}
	stopTime := time.Now().Add(time.Minute * 15)
	updateCompleted := false
	for ; time.Until(stopTime) > 0; cpuSharer.Sleep(time.Second) {
		if syncedImage, err := h.waitLastImageName(cpuSharer); err != nil {
			return err
		} else if syncedImage == imageName {
			updateCompleted = true
			break
		}
	}
	if !updateCompleted {
		return errors.New("timed out waiting for image update to complete")
	}
	h.logger.Debugln(0, "upgraded")
	cpuSharer.Sleep(time.Second * 15)
	list, _, err = h.getFailingHealthChecks(cpuSharer, time.Minute)
	if err != nil {
		return err
	} else {
		for _, entry := range list {
			if _, ok := h.initialUnhealthyList[entry]; !ok {
				return fmt.Errorf("health check failed: %s:", entry)
			}
		}
	}
	h.logger.Debugln(0, "still healthy")
	return nil
}

func (h *hypervisorType) waitLastImageName(cpuSharer *cpusharer.FifoCpuSharer) (
	string, error) {
	stopTime := time.Now().Add(time.Minute)
	for ; time.Until(stopTime) > 0; cpuSharer.Sleep(time.Second * 5) {
		imageName, err := h.getLastImageName(cpuSharer)
		if err != nil {
			h.logger.Debugln(0, err)
			continue
		}
		return imageName, nil
	}
	return "", errors.New("timed out getting last image name")
}
