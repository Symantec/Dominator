package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

type vmExporterVirsh struct{}

func exportVirshVmSubcommand(args []string, logger log.DebugLogger) error {
	if err := exportVirshVm(args[0], logger); err != nil {
		return fmt.Errorf("Error exporting VM: %s", err)
	}
	return nil
}

func exportVirshVm(vmHostname string, logger log.DebugLogger) error {
	return vmExport(vmHostname, vmExporterVirsh{}, logger)
}

func (exporter vmExporterVirsh) createVm(hostname string,
	vmInfo proto.ExportLocalVmInfo) error {
	virshInfo := virshInfoType{
		Cpu: cpuType{Mode: "host-passthrough"},
		Devices: devicesInfo{
			Interfaces: []interfaceType{
				addressToInterface(vmInfo.Address, vmInfo.Bridges[0])},
			SerialPorts: []serialType{{
				Type:   "file",
				Source: serialSourceType{Path: "/dev/null"},
			}},
		},
		Memory: memoryInfo{Value: vmInfo.MemoryInMiB, Unit: "MiB"},
		Name:   hostname,
		Os: osInfo{
			Type: osTypeInfo{Arch: "x86_64", Machine: "pc", Value: "hvm"},
		},
		Type: "kvm",
		VCpu: vCpuInfo{
			Num:       (vmInfo.MilliCPUs + 500) / 1000,
			Placement: "static"},
	}
	for index, address := range vmInfo.SecondaryAddresses {
		virshInfo.Devices.Interfaces = append(virshInfo.Devices.Interfaces,
			addressToInterface(address, vmInfo.Bridges[index+1]))
	}
	for index, volume := range vmInfo.Volumes {
		volume, err := makeVolume(volume, index,
			vmInfo.VolumeLocations[index].Filename)
		if err != nil {
			return err
		}
		virshInfo.Devices.Volumes = append(virshInfo.Devices.Volumes, volume)
	}
	if virshInfo.VCpu.Num < 1 {
		virshInfo.VCpu.Num = 1
	}
	xmlData, err := xml.MarshalIndent(virshInfo, "", "    ")
	if err != nil {
		return err
	}
	xmlData = append(xmlData, '\n')
	os.Stdout.Write(xmlData)
	response, err := askForInputChoice(
		fmt.Sprintf("Have you added %s/%s to your DHCP Server",
			vmInfo.Address.MacAddress, vmInfo.Address.IpAddress),
		[]string{"yes", "no"})
	if err != nil {
		return err
	}
	switch response {
	case "no":
		return errors.New("DHCP not configured for VM")
	case "yes":
	default:
		return fmt.Errorf("invalid response: %s", response)
	}
	tmpdir, err := ioutil.TempDir("", "export-vm")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)
	tmpfile := filepath.Join(tmpdir, hostname+".xml")
	if file, err := os.Create(tmpfile); err != nil {
		return err
	} else {
		defer file.Close()
		if _, err := file.Write(xmlData); err != nil {
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
	}
	cmd := exec.Command("virsh", "define", tmpfile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error defining virsh VM: %s", err)
	}
	cmd = exec.Command("virsh", "start", hostname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error starting virsh VM: %s", err)
	}
	return nil
}

func (exporter vmExporterVirsh) destroyVm(hostname string) error {
	cmd := exec.Command("virsh", "destroy", hostname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error destroying new VM: %s", err)
	}
	cmd = exec.Command("virsh",
		[]string{"undefine", "--managed-save", "--snapshots-metadata",
			"--remove-all-storage", hostname}...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error undefining new VM: %s", err)
	}
	return nil
}

func addressToInterface(address proto.Address, bridge string) interfaceType {
	return interfaceType{
		Mac:    macType{Address: address.MacAddress},
		Model:  modelType{Type: "virtio"},
		Source: bridgeType{Bridge: bridge},
		Type:   "bridge",
	}
}

func makeVolume(volume proto.Volume, index int,
	filename string) (volumeType, error) {
	dirname := filepath.Dir(filename)
	dirname = filepath.Join(filepath.Dir(dirname), "export",
		filepath.Base(dirname))
	if err := os.MkdirAll(dirname, fsutil.DirPerms); err != nil {
		return volumeType{}, err
	}
	exportFilename := filepath.Join(dirname, filepath.Base(filename))
	os.Remove(exportFilename)
	if err := os.Link(filename, exportFilename); err != nil {
		return volumeType{}, err
	}
	return volumeType{
		Device: "disk",
		Driver: driverType{
			Name:  "qemu",
			Type:  volume.Format.String(),
			Cache: "none",
			Io:    "native",
		},
		Source: sourceType{File: exportFilename},
		Target: targetType{Device: "vd" + string('a'+index), Bus: "virtio"},
		Type:   "file",
	}, nil
}
