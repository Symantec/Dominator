package main

import (
	"encoding/xml"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/srpc"
	//"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type devicesInfo struct {
	Interfaces []interfaceType `xml:"interface"`
	Volumes    []volumeType    `xml:"disk"`
}

type driverType struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

type interfaceType struct {
	Mac  macType `xml:"mac"`
	Type string  `xml:"type,attr"`
}

type macType struct {
	Address string `xml:"address,attr"`
}

type memoryInfo struct {
	Value uint64 `xml:",chardata"`
	Unit  string `xml:"unit,attr"`
}

type sourceType struct {
	File string `xml:"file,attr"`
}

type vCpuInfo struct {
	Num       uint   `xml:",chardata"`
	Placement string `xml:"placement,attr"`
}

type virshInfoType struct {
	Devices devicesInfo `xml:"devices"`
	Memory  memoryInfo  `xml:"memory"`
	Name    string      `xml:"name"`
	VCpu    vCpuInfo    `xml:"vcpu"`
}

type volumeType struct {
	Device string     `xml:"device,attr"`
	Driver driverType `xml:"driver"`
	Source sourceType `xml:"source"`
	Type   string     `xml:"type,attr"`
}

func importVirshVmSubcommand(args []string, logger log.DebugLogger) {
	if err := importVirshVm(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error importing VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func importVirshVm(domainName string, logger log.DebugLogger) error {
	ipList, err := net.LookupIP(domainName)
	if err != nil {
		return err
	}
	if len(ipList) != 1 {
		return fmt.Errorf("number of IPs %d != 1", len(ipList))
	}
	tags := vmTags.Copy()
	if _, ok := tags["Name"]; !ok {
		tags["Name"] = domainName
	}
	request := proto.ImportLocalVmRequest{VmInfo: proto.VmInfo{
		Hostname:    domainName,
		OwnerGroups: ownerGroups,
		OwnerUsers:  ownerUsers,
		Tags:        tags,
	}}
	request.VerificationCookie, err = readImportCookie(logger)
	if err != nil {
		return err
	}
	hypervisor := fmt.Sprintf(":%d", *hypervisorPortNum)
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	directories, err := listVolumeDirectories(client)
	if err != nil {
		return err
	}
	volumeRoots := make(map[string]string, len(directories))
	for _, dirname := range directories {
		volumeRoots[filepath.Dir(dirname)] = dirname
	}
	cmd := exec.Command("virsh", []string{"domstate", domainName}...)
	stdout, err := cmd.Output()
	if err != nil {
		return err
	}
	state := strings.TrimSpace(string(stdout))
	if state != "shut off" {
		return fmt.Errorf("domain must be shut off but is \"%s\"", state)
	}
	cmd = exec.Command("virsh",
		[]string{"dumpxml", "--inactive", domainName}...)
	stdout, err = cmd.Output()
	if err != nil {
		return err
	}
	var virshInfo virshInfoType
	if err := xml.Unmarshal(stdout, &virshInfo); err != nil {
		return err
	}
	json.WriteWithIndent(os.Stdout, "    ", virshInfo)
	if numIf := len(virshInfo.Devices.Interfaces); numIf != 1 {
		return fmt.Errorf("number of interfaces %d != 1", numIf)
	}
	request.VmInfo.Address = proto.Address{
		IpAddress:  ipList[0],
		MacAddress: virshInfo.Devices.Interfaces[0].Mac.Address,
	}
	switch virshInfo.Memory.Unit {
	case "KiB":
		request.VmInfo.MemoryInMiB = virshInfo.Memory.Value >> 10
	case "MiB":
		request.VmInfo.MemoryInMiB = virshInfo.Memory.Value
	case "GiB":
		request.VmInfo.MemoryInMiB = virshInfo.Memory.Value << 10
	default:
		return fmt.Errorf("unknown memory unit: %s", virshInfo.Memory.Unit)
	}
	request.VmInfo.MilliCPUs = virshInfo.VCpu.Num * 1000
	myPidStr := strconv.Itoa(os.Getpid())
	for index, inputVolume := range virshInfo.Devices.Volumes {
		if inputVolume.Device != "disk" {
			continue
		}
		var volumeFormat proto.VolumeFormat
		err := volumeFormat.UnmarshalText([]byte(inputVolume.Driver.Type))
		if err != nil {
			return err
		}
		inputFilename := inputVolume.Source.File
		var volumeRoot string
		for dirname := filepath.Dir(inputFilename); ; {
			if vr, ok := volumeRoots[dirname]; ok {
				volumeRoot = vr
				break
			}
			if dirname == "/" {
				break
			}
			dirname = filepath.Dir(dirname)
		}
		if volumeRoot == "" {
			return fmt.Errorf("no Hypervisor directory for: %s", inputFilename)
		}
		outputDirname := filepath.Join(volumeRoot, "import", myPidStr)
		if err := os.MkdirAll(outputDirname, dirPerms); err != nil {
			return err
		}
		defer os.RemoveAll(outputDirname)
		outputFilename := filepath.Join(outputDirname,
			fmt.Sprintf("volume-%d", index))
		if err := os.Link(inputFilename, outputFilename); err != nil {
			return err
		}
		request.VolumeFilenames = append(request.VolumeFilenames,
			outputFilename)
		request.VmInfo.Volumes = append(request.VmInfo.Volumes,
			proto.Volume{Format: volumeFormat})
	}
	json.WriteWithIndent(os.Stdout, "    ", request)
	var reply proto.GetVmInfoResponse
	err = client.RequestReply("Hypervisor.ImportLocalVm", request, &reply)
	if err != nil {
		return err
	}
	if err := errors.New(reply.Error); err != nil {
		return err
	}
	logger.Debugln(0, "imported VM")
	for _, dirname := range directories {
		os.RemoveAll(filepath.Join(dirname, "import", myPidStr))
	}
	if err := maybeWatchVm(client, hypervisor, ipList[0], logger); err != nil {
		return err
	}
	if err := askForCommitDecision(client, ipList[0]); err != nil {
		return err
	}
	defer virshInfo.deleteVolumes()
	cmd = exec.Command("virsh",
		[]string{"undefine", "--managed-save", "--snapshots-metadata",
			"--remove-all-storage", domainName}...)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Println(string(output))
		return err
	}
	return nil
}

func (virshInfo virshInfoType) deleteVolumes() {
	for _, inputVolume := range virshInfo.Devices.Volumes {
		if inputVolume.Device != "disk" {
			continue
		}
		os.Remove(inputVolume.Source.File)
	}
}
