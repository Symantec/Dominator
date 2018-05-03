// +build go1.10

package metadatad

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/log/prefixlogger"
	"github.com/Symantec/Dominator/lib/wsyscall"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type statusType struct {
	namespaceFd int
	threadId    int
	err         error
}

func (s *server) startServer() error {
	cmd := exec.Command("ebtables", "-t", "nat", "-F")
	if err := cmd.Run(); err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, bridge := range s.bridges {
		if err := s.startServerOnBridge(bridge); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) startServerOnBridge(bridge net.Interface) error {
	logger := prefixlogger.New(bridge.Name+": ", s.logger)
	startChannel := make(chan struct{})
	statusChannel := make(chan statusType, 1)
	go s.createNamespace(startChannel, statusChannel, logger)
	status := <-statusChannel
	if status.err != nil {
		return status.err
	}
	if err := createInterface(bridge, status.threadId, logger); err != nil {
		return err
	}
	startChannel <- struct{}{}
	status = <-statusChannel
	if status.err != nil {
		return status.err
	}
	subnetChannel := s.manager.MakeSubnetChannel()
	go s.addSubnets(status.namespaceFd, subnetChannel, logger)
	return nil
}

func (s *server) addSubnets(namespaceFd int, subnetChannel <-chan proto.Subnet,
	logger log.DebugLogger) {
	if err := wsyscall.SetNetNamespace(namespaceFd); err != nil {
		logger.Println(err)
		return
	}
	for subnet := range subnetChannel {
		for _, bridge := range s.bridges {
			addRouteForBridge(bridge, subnet, logger)
		}
	}
}

func addRouteForBridge(bridge net.Interface, subnet proto.Subnet,
	logger log.DebugLogger) {
	subnetMask := net.IPMask(subnet.IpMask)
	subnetAddr := subnet.IpGateway.Mask(subnetMask)
	addr := subnetAddr.String()
	mask := fmt.Sprintf("%d.%d.%d.%d",
		subnetMask[0], subnetMask[1], subnetMask[2], subnetMask[3])
	cmd := exec.Command("route", "add", "-net", addr, "netmask", mask, "eth0")
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Printf("error adding route: %s/%s: %s: %s",
			addr, mask, err, string(output))
	} else {
		logger.Debugf(0, "added route: %s/%s\n", addr, mask)
	}
}

func (s *server) createNamespace(startChannel <-chan struct{},
	statusChannel chan<- statusType, logger log.DebugLogger) {
	namespaceFd, threadId, err := wsyscall.UnshareNetNamespace()
	if err != nil {
		statusChannel <- statusType{err: err}
		return
	}
	statusChannel <- statusType{namespaceFd: namespaceFd, threadId: threadId}
	<-startChannel
	cmd := exec.Command("ifconfig", "eth0", "169.254.169.254", "netmask",
		"255.255.255.255", "up")
	if err := cmd.Run(); err != nil {
		statusChannel <- statusType{err: err}
		return
	}
	hypervisorListener, err := net.Listen("tcp",
		fmt.Sprintf("169.254.169.254:%d", s.hypervisorPortNum))
	if err != nil {
		statusChannel <- statusType{err: err}
		return
	}
	metadataListener, err := net.Listen("tcp", "169.254.169.254:80")
	if err != nil {
		statusChannel <- statusType{err: err}
		return
	}
	statusChannel <- statusType{namespaceFd: namespaceFd, threadId: threadId}
	logger.Printf("starting metadata server in thread: %d\n", threadId)
	go http.Serve(hypervisorListener, nil)
	http.Serve(metadataListener, s)
}

func createInterface(bridge net.Interface, threadId int,
	logger log.DebugLogger) error {
	localName := bridge.Name + "-ll"
	remoteName := bridge.Name + "-lr"
	cmd := exec.Command("ip", "link", "add", localName, "type", "veth",
		"peer", "name", remoteName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error creating veth for bridge: %s: %s: %s",
			bridge.Name, err, output)
	}
	cmd = exec.Command("ifconfig", localName, "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error bringing up local interface: %s: %s: %s",
			localName, err, output)
	}
	remoteInterface, err := net.InterfaceByName(remoteName)
	if err != nil {
		return err
	}
	cmd = exec.Command("ip", "link", "set", remoteName, "netns",
		strconv.FormatInt(int64(threadId), 10), "name", "eth0")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error moving interface to namespace: %s: %s: %s",
			remoteName, err, output)
	}
	cmd = exec.Command("ip", "link", "set", localName, "master", bridge.Name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error adding interface: %s to bridge: %s: %s: %s",
			localName, bridge.Name, err, output)
	}
	hwAddr := remoteInterface.HardwareAddr.String()
	cmd = exec.Command("ebtables", "-t", "nat", "-A", "PREROUTING",
		"--logical-in", bridge.Name, "-p", "ip",
		"--ip-dst", "169.254.0.0/16", "-j", "dnat", "--to-destination",
		hwAddr)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"error adding ebtables dnat to: %s to bridge: %s: %s: %s",
			hwAddr, bridge.Name, err, output)
	}
	logger.Printf("created veth, remote addr: %s\n", hwAddr)
	return nil
}
