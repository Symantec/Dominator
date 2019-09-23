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
	"time"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type bridgeType struct {
	Bridge string `xml:"bridge,attr"`
}

type cpuType struct {
	Mode string `xml:"mode,attr"`
}

type devicesInfo struct {
	Volumes     []volumeType    `xml:"disk"`
	Interfaces  []interfaceType `xml:"interface"`
	SerialPorts []serialType    `xml:"serial"`
}

type driverType struct {
	Name  string `xml:"name,attr"`
	Type  string `xml:"type,attr"`
	Cache string `xml:"cache,attr"`
	Io    string `xml:"io,attr"`
}

type interfaceType struct {
	Mac    macType    `xml:"mac"`
	Model  modelType  `xml:"model"`
	Source bridgeType `xml:"source"`
	Type   string     `xml:"type,attr"`
}

type macType struct {
	Address string `xml:"address,attr"`
}

type memoryInfo struct {
	Value uint64 `xml:",chardata"`
	Unit  string `xml:"unit,attr"`
}

type modelType struct {
	Type string `xml:"type,attr"`
}

type osInfo struct {
	Type osTypeInfo `xml:"type"`
}

type osTypeInfo struct {
	Arch    string `xml:"arch,attr"`
	Machine string `xml:"machine,attr"`
	Value   string `xml:",chardata"`
}

type serialType struct {
	Source serialSourceType `xml:"source"`
	Type   string           `xml:"type,attr"`
}

type serialSourceType struct {
	Path string `xml:"path,attr"`
}

type sourceType struct {
	File string `xml:"file,attr"`
}

type targetType struct {
	Device string `xml:"dev,attr"`
	Bus    string `xml:"bus,attr"`
}

type vCpuInfo struct {
	Num       uint   `xml:",chardata"`
	Placement string `xml:"placement,attr"`
}

type virshInfoType struct {
	XMLName xml.Name    `xml:"domain"`
	Cpu     cpuType     `xml:"cpu"`
	Devices devicesInfo `xml:"devices"`
	Memory  memoryInfo  `xml:"memory"`
	Name    string      `xml:"name"`
	Os      osInfo      `xml:"os"`
	Type    string      `xml:"type,attr"`
	VCpu    vCpuInfo    `xml:"vcpu"`
}

type volumeType struct {
	Device string     `xml:"device,attr"`
	Driver driverType `xml:"driver"`
	Source sourceType `xml:"source"`
	Target targetType `xml:"target"`
	Type   string     `xml:"type,attr"`
}

func importVirshVmSubcommand(args []string, logger log.DebugLogger) error {
	macAddr := args[0]
	domainName := args[1]
	args = args[2:]
	if len(args)%2 != 0 {
		return fmt.Errorf("missing IP address for MAC: %s", args[len(args)-1])
	}
	sAddrs := make([]proto.Address, 0, len(args)/2)
	for index := 0; index < len(args); index += 2 {
		ipAddr := args[index+1]
		ipList, err := net.LookupIP(ipAddr)
		if err != nil {
			return err
		}
		if len(ipList) != 1 {
			return fmt.Errorf("number of IPs for %s: %d != 1",
				ipAddr, len(ipList))
		}
		sAddrs = append(sAddrs, proto.Address{
			IpAddress:  ipList[0],
			MacAddress: args[index],
		})
	}
	if err := importVirshVm(macAddr, domainName, sAddrs, logger); err != nil {
		return fmt.Errorf("Error importing VM: %s", err)
	}
	return nil
}

func ensureDomainIsStopped(domainName string) error {
	state, err := getDomainState(domainName)
	if err != nil {
		return err
	}
	if state == "shut off" {
		return nil
	}
	if state != "running" {
		return fmt.Errorf("domain is in unsupported state \"%s\"", state)
	}
	response, err := askForInputChoice("Cannot import running VM",
		[]string{"shutdown", "quit"})
	if err != nil {
		return err
	}
	if response == "quit" {
		return fmt.Errorf("domain must be shut off but is \"%s\"", state)
	}
	err = exec.Command("virsh", []string{"shutdown", domainName}...).Run()
	if err != nil {
		return fmt.Errorf("error shutting down VM: %s", err)
	}
	for ; ; time.Sleep(time.Second) {
		state, err := getDomainState(domainName)
		if err != nil {
			if strings.Contains(err.Error(), "Domain not found") {
				return nil
			}
			return err
		}
		if state == "shut off" {
			return nil
		}
	}
}

func getDomainState(domainName string) (string, error) {
	cmd := exec.Command("virsh", []string{"domstate", domainName}...)
	stdout, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting VM status: %s",
			err.(*exec.ExitError).Stderr)
	}
	return strings.TrimSpace(string(stdout)), nil
}

func importVirshVm(macAddr, domainName string, sAddrs []proto.Address,
	logger log.DebugLogger) error {
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
		ConsoleType:   consoleType,
		DisableVirtIO: *disableVirtIO,
		Hostname:      domainName,
		OwnerGroups:   ownerGroups,
		OwnerUsers:    ownerUsers,
		Tags:          tags,
	}}
	hypervisor := fmt.Sprintf(":%d", *hypervisorPortNum)
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	verificationCookie, err := readRootCookie(client, logger)
	if err != nil {
		return err
	}
	directories, err := listVolumeDirectories(client)
	if err != nil {
		return err
	}
	volumeRoots := make(map[string]string, len(directories))
	for _, dirname := range directories {
		volumeRoots[filepath.Dir(dirname)] = dirname
	}
	cmd := exec.Command("virsh",
		[]string{"dumpxml", "--inactive", domainName}...)
	stdout, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("error getting XML data: %s", err)
	}
	var virshInfo virshInfoType
	if err := xml.Unmarshal(stdout, &virshInfo); err != nil {
		return err
	}
	json.WriteWithIndent(os.Stdout, "    ", virshInfo)
	if numIf := len(virshInfo.Devices.Interfaces); numIf != len(sAddrs)+1 {
		return fmt.Errorf("number of interfaces %d != %d",
			numIf, len(sAddrs)+1)
	}
	if macAddr != virshInfo.Devices.Interfaces[0].Mac.Address {
		return fmt.Errorf("MAC address specified: %s != virsh data: %s",
			macAddr, virshInfo.Devices.Interfaces[0].Mac.Address)
	}
	request.VmInfo.Address = proto.Address{
		IpAddress:  ipList[0],
		MacAddress: virshInfo.Devices.Interfaces[0].Mac.Address,
	}
	for index, sAddr := range sAddrs {
		if sAddr.MacAddress !=
			virshInfo.Devices.Interfaces[index+1].Mac.Address {
			return fmt.Errorf("MAC address specified: %s != virsh data: %s",
				sAddr.MacAddress,
				virshInfo.Devices.Interfaces[index+1].Mac.Address)
		}
		request.SecondaryAddresses = append(request.SecondaryAddresses, sAddr)
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
	if err := ensureDomainIsStopped(domainName); err != nil {
		return err
	}
	logger.Debugln(0, "finding volumes")
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
	request.VerificationCookie = verificationCookie
	var reply proto.GetVmInfoResponse
	logger.Debugln(0, "issuing import RPC")
	err = client.RequestReply("Hypervisor.ImportLocalVm", request, &reply)
	if err != nil {
		return fmt.Errorf("Hypervisor.ImportLocalVm RPC failed: %s", err)
	}
	if err := errors.New(reply.Error); err != nil {
		return fmt.Errorf("Hypervisor failed to import: %s", err)
	}
	logger.Debugln(0, "imported VM")
	for _, dirname := range directories {
		os.RemoveAll(filepath.Join(dirname, "import", myPidStr))
	}
	if err := maybeWatchVm(client, hypervisor, ipList[0], logger); err != nil {
		return err
	}
	if err := askForCommitDecision(client, ipList[0]); err != nil {
		if err == errorCommitAbandoned {
			response, _ := askForInputChoice(
				"Do you want to restart the old VM", []string{"y", "n"})
			if response != "y" {
				return err
			}
			cmd = exec.Command("virsh", "start", domainName)
			if output, err := cmd.CombinedOutput(); err != nil {
				logger.Println(string(output))
				return err
			}
		}
		return err
	}
	defer virshInfo.deleteVolumes()
	cmd = exec.Command("virsh",
		[]string{"undefine", "--managed-save", "--snapshots-metadata",
			"--remove-all-storage", domainName}...)
	if output, err := cmd.CombinedOutput(); err != nil {
		logger.Println(string(output))
		return fmt.Errorf("error destroying old VM: %s", err)
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
