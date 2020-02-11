package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"time"

	hyperclient "github.com/Cloud-Foundations/Dominator/hypervisor/client"
	imgclient "github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/image"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	fm_proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
	subclient "github.com/Cloud-Foundations/Dominator/sub/client"
)

func diffSubcommand(args []string, logger log.DebugLogger) error {
	return diffTypedImages(args[0], args[1], args[2])
}

func diffTypedImages(tool string, lName string, rName string) error {
	lfs, err := getTypedImage(lName)
	if err != nil {
		return fmt.Errorf("Error getting left image: %s", err)
	}
	if lfs, err = applyDeleteFilter(lfs); err != nil {
		return fmt.Errorf("Error filtering left image: %s", err)
	}
	rfs, err := getTypedImage(rName)
	if err != nil {
		return fmt.Errorf("Error getting right image: %s", err)
	}
	if rfs, err = applyDeleteFilter(rfs); err != nil {
		return fmt.Errorf("Error filtering right image: %s", err)
	}
	err = diffImages(tool, lfs, rfs)
	if err != nil {
		return fmt.Errorf("Error diffing images: %s", err)
	}
	return nil
}

func getTypedImage(typedName string) (*filesystem.FileSystem, error) {
	if len(typedName) < 3 || typedName[1] != ':' {
		imageSClient, _ := getClients()
		return getFsOfImage(imageSClient, typedName)
	}
	switch name := typedName[2:]; typedName[0] {
	case 'd':
		return scanDirectory(name)
	case 'f':
		return readFileSystem(name)
	case 'i':
		imageSClient, _ := getClients()
		return getFsOfImage(imageSClient, name)
	case 'l':
		return readFsOfImage(name)
	case 's':
		return pollImage(name)
	case 'v':
		return scanVm(name)
	default:
		return nil, errors.New("unknown image type: " + typedName[:1])
	}
}

func scanDirectory(name string) (*filesystem.FileSystem, error) {
	fs, err := buildImageWithHasher(nil, nil, name, nil)
	if err != nil {
		return nil, err
	}
	return fs, nil
}

func readFileSystem(name string) (*filesystem.FileSystem, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var fileSystem filesystem.FileSystem
	if err := gob.NewDecoder(file).Decode(&fileSystem); err != nil {
		return nil, err
	}
	fileSystem.RebuildInodePointers()
	return &fileSystem, nil
}

func getImage(client *srpc.Client, name string) (*image.Image, error) {
	img, err := imgclient.GetImageWithTimeout(client, name, *timeout)
	if err != nil {
		return nil, err
	}
	if img == nil {
		return nil, errors.New(name + ": not found")
	}
	img.FileSystem.RebuildInodePointers()
	return img, nil
}

func getFsOfImage(client *srpc.Client, name string) (
	*filesystem.FileSystem, error) {
	if image, err := getImage(client, name); err != nil {
		return nil, err
	} else {
		return image.FileSystem, nil
	}
}

func readFsOfImage(name string) (*filesystem.FileSystem, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var image image.Image
	if err := gob.NewDecoder(file).Decode(&image); err != nil {
		return nil, err
	}
	image.FileSystem.RebuildInodePointers()
	return image.FileSystem, nil
}

func pollImage(name string) (*filesystem.FileSystem, error) {
	clientName := fmt.Sprintf("%s:%d", name, constants.SubPortNumber)
	srpcClient, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		return nil, fmt.Errorf("Error dialing %s", err)
	}
	defer srpcClient.Close()
	var request sub.PollRequest
	var reply sub.PollResponse
	if err = subclient.CallPoll(srpcClient, request, &reply); err != nil {
		return nil, err
	}
	if reply.FileSystem == nil {
		return nil, errors.New("no poll data")
	}
	reply.FileSystem.RebuildInodePointers()
	return reply.FileSystem, nil
}

func scanVm(name string) (*filesystem.FileSystem, error) {
	vmIpAddr, srpcClient, err := getVmIpAndHypervisor(name)
	if err != nil {
		return nil, err
	}
	defer srpcClient.Close()
	fs, err := hyperclient.ScanVmRoot(srpcClient, vmIpAddr, nil)
	if err != nil {
		return nil, err
	}
	fs.RebuildInodePointers()
	return fs, nil
}

func getVmIpAndHypervisor(vmHostname string) (net.IP, *srpc.Client, error) {
	vmIpAddr, err := lookupIP(vmHostname)
	if err != nil {
		return nil, nil, err
	}
	hypervisorAddress, err := findHypervisor(vmIpAddr)
	if err != nil {
		return nil, nil, err
	}
	client, err := srpc.DialHTTP("tcp", hypervisorAddress, time.Second*10)
	if err != nil {
		return nil, nil, err
	}
	return vmIpAddr, client, nil
}

func findHypervisor(vmIpAddr net.IP) (string, error) {
	if *hypervisorHostname != "" {
		return fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum),
			nil
	} else if *fleetManagerHostname != "" {
		fm := fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum)
		client, err := srpc.DialHTTP("tcp", fm, time.Second*10)
		if err != nil {
			return "", err
		}
		defer client.Close()
		return findHypervisorClient(client, vmIpAddr)
	} else {
		return fmt.Sprintf("localhost:%d", *hypervisorPortNum), nil
	}
}

func findHypervisorClient(client *srpc.Client,
	vmIpAddr net.IP) (string, error) {
	request := fm_proto.GetHypervisorForVMRequest{vmIpAddr}
	var reply fm_proto.GetHypervisorForVMResponse
	err := client.RequestReply("FleetManager.GetHypervisorForVM", request,
		&reply)
	if err != nil {
		return "", err
	}
	if err := errors.New(reply.Error); err != nil {
		return "", err
	}
	return reply.HypervisorAddress, nil
}

func lookupIP(vmHostname string) (net.IP, error) {
	if ips, err := net.LookupIP(vmHostname); err != nil {
		return nil, err
	} else if len(ips) != 1 {
		return nil, fmt.Errorf("num IPs: %d != 1", len(ips))
	} else {
		return ips[0], nil
	}
}

func diffImages(tool string, lfs, rfs *filesystem.FileSystem) error {
	lname, err := writeImage(lfs)
	defer os.Remove(lname)
	if err != nil {
		return err
	}
	rname, err := writeImage(rfs)
	defer os.Remove(rname)
	if err != nil {
		return err
	}
	cmd := exec.Command(tool, lname, rname)
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func writeImage(fs *filesystem.FileSystem) (string, error) {
	file, err := ioutil.TempFile("", "imagetool")
	if err != nil {
		return "", err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()
	return file.Name(), fs.Listf(writer, listSelector, listFilter)
}
