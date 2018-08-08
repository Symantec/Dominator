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
	"syscall"
	"time"

	imageclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/Symantec/Dominator/lib/rpcclientpool"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/tags"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
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
	initialTags               tags.Tags
	initialUnhealthyList      map[string]struct{}
	logger                    log.DebugLogger
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
	if *topologyDir != "" {
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
	if foundImage, err := checkImage(imageName); err != nil {
		return err
	} else if !foundImage {
		return fmt.Errorf("image: %s not found", imageName)
	}
	fleetManagerClientResource := srpc.NewClientResource("tcp",
		fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum))
	defer fleetManagerClientResource.ScheduleClose()
	hypervisorAddresses, err := listGoodHypervisors(fleetManagerClientResource)
	if err != nil {
		return err
	}
	hypervisors := make([]*hypervisorType, 0, len(hypervisorAddresses))
	defer closeHypervisors(hypervisors)
	tagsForHypervisors, err := getTagsForHypervisors(fleetManagerClientResource)
	if err != nil {
		return fmt.Errorf("failure getting tags: %s", err)
	}
	for _, address := range hypervisorAddresses {
		if hostname, _, err := net.SplitHostPort(address); err != nil {
			return err
		} else {
			logger := prefixlogger.New(hostname+": ", logger)
			tgs := tagsForHypervisors[hostname]
			currentRequiredImage := tgs["RequiredImage"]
			if currentRequiredImage != "" &&
				path.Dir(currentRequiredImage) != path.Dir(imageName) {
				logger.Printf(
					"image stream: current=%s != new=%s, skipping\n",
					path.Dir(currentRequiredImage), path.Dir(imageName))
				continue
			}
			hypervisor := &hypervisorType{
				healthAgentClientResource: rpcclientpool.New("tcp",
					fmt.Sprintf("%s:%d", hostname, 6910), true, ""),
				hostname:             hostname,
				initialTags:          tgs,
				initialUnhealthyList: make(map[string]struct{}),
				logger:               logger,
				subClientResource: srpc.NewClientResource("tcp",
					fmt.Sprintf("%s:%d", hostname, constants.SubPortNumber)),
			}
			if lastImage, err := hypervisor.getLastImageName(); err != nil {
				logger.Printf("skipping %s: %s\n", hostname, err)
			} else if lastImage == imageName {
				logger.Printf("%s already updated, skipping\n", hostname)
			} else {
				err := hypervisor.updateTagForHypervisor(
					fleetManagerClientResource, "PlannedImage", imageName)
				if err != nil {
					return fmt.Errorf("%s: failure updating tags: %s",
						hostname, err)
				}
				hypervisors = append(hypervisors, hypervisor)
			}
		}
	}
	for _, hypervisor := range hypervisors {
		if list, _, err := hypervisor.getFailingHealthChecks(); err != nil {
			hypervisor.logger.Println(err)
			continue
		} else if len(list) > 0 {
			for _, failed := range list {
				hypervisor.initialUnhealthyList[failed] = struct{}{}
			}
		}
		err := hypervisor.upgrade(fleetManagerClientResource, imageName)
		if err != nil {
			return fmt.Errorf("error upgrading: %s: %s",
				hypervisor.hostname, err)
		}
	}
	if *topologyDir != "" {
		var tgs tags.Tags
		tagsFilename := filepath.Join(*topologyDir, *location, "tags.json")
		if err := json.ReadFromFile(tagsFilename, &tgs); err != nil {
			return err
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
			locationInsert = "in " + *location
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
	client, err := clientResource.GetHTTP(nil, 0)
	if err != nil {
		return nil, err
	}
	defer client.Put()
	request := fm_proto.ListHypervisorsInLocationRequest{Location: *location}
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

func (h *hypervisorType) getFailingHealthChecks() ([]string, time.Time, error) {
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
		client.Close()
		return nil, time.Time{}, errors.New("list metric is not []string")
	} else {
		if timestamp, ok := metric.TimeStamp.(time.Time); ok {
			return list, timestamp, nil
		} else {
			return list, time.Time{}, nil
		}
	}
}

func (h *hypervisorType) getLastImageName() (string, error) {
	stopTime := time.Now().Add(time.Minute * 5)
	for ; time.Until(stopTime) > 0; time.Sleep(time.Second) {
		client, err := h.subClientResource.GetHTTP(nil, 0)
		if err != nil {
			h.logger.Debugln(0, err)
			continue
		}
		request := sub_proto.PollRequest{ShortPollOnly: true}
		var reply sub_proto.PollResponse
		if err := subclient.CallPoll(client, request, &reply); err != nil {
			client.Close()
			if err != io.EOF {
				h.logger.Debugln(0, err)
			}
			continue
		}
		client.Put()
		return reply.LastSuccessfulImageName, nil
	}
	return "", errors.New("timed out getting last image name")
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
	imageName string) error {
	h.logger.Debugln(0, "upgrading")
	err := h.updateTagForHypervisor(clientResource, "RequiredImage", imageName)
	if err != nil {
		return err
	}
	stopTime := time.Now().Add(time.Minute * 15)
	updateCompleted := false
	for ; time.Until(stopTime) > 0; time.Sleep(time.Second) {
		if syncedImage, err := h.getLastImageName(); err != nil {
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
	time.Sleep(time.Second * 15)
	if list, _, err := h.getFailingHealthChecks(); err != nil {
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
