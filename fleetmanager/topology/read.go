package topology

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"

	"github.com/Symantec/Dominator/lib/json"
)

func cloneSet(set map[string]struct{}) map[string]struct{} {
	clone := make(map[string]struct{}, len(set))
	for key := range set {
		clone[key] = struct{}{}
	}
	return clone
}

func load(topologyDir string) (*Topology, error) {
	topology := &Topology{machineParents: make(map[string]*Directory)}
	directory, err := topology.readDirectory(topologyDir, "",
		make(map[string]struct{}))
	if err != nil {
		return nil, err
	}
	topology.Root = directory
	return topology, nil
}

func loadMachines(filename string) ([]*Machine, error) {
	var machines []*Machine
	if err := json.ReadFromFile(filename, &machines); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error reading: %s: %s", filename, err)
	}
	for _, machine := range machines {
		if len(machine.HostIpAddress) == 0 {
			if addrs, err := net.LookupIP(machine.Hostname); err != nil {
				return nil, err
			} else if len(addrs) != 1 {
				return nil, fmt.Errorf("num addresses for: %s: %d!=1",
					machine.Hostname, len(addrs))
			} else {
				machine.HostIpAddress = addrs[0]
			}
		}
		if len(machine.HostIpAddress) == 16 {
			machine.HostIpAddress = machine.HostIpAddress.To4()
		}
	}
	return machines, nil
}

func loadSubnets(filename string) ([]*Subnet, error) {
	var subnets []*Subnet
	if err := json.ReadFromFile(filename, &subnets); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("error reading: %s: %s", filename, err)
	}
	gatewayIPs := make(map[string]struct{}, len(subnets))
	for _, subnet := range subnets {
		subnet.Shrink()
		gatewayIp := subnet.IpGateway.String()
		if _, ok := gatewayIPs[gatewayIp]; ok {
			return nil, fmt.Errorf("duplicate gateway IP: %s", gatewayIp)
		} else {
			gatewayIPs[gatewayIp] = struct{}{}
		}
		subnet.reservedIpAddrs = make(map[string]struct{})
		for _, ipAddr := range subnet.ReservedIPs {
			subnet.reservedIpAddrs[ipAddr.String()] = struct{}{}
		}
	}
	return subnets, nil
}

func (t *Topology) readDirectory(topDir, dirname string,
	subnetIds map[string]struct{}) (*Directory, error) {
	directory := &Directory{
		nameToDirectory:  make(map[string]*Directory),
		path:             dirname,
		subnetIdToSubnet: make(map[string]*Subnet),
	}
	dirpath := filepath.Join(topDir, dirname)
	if err := t.loadMachines(directory, dirpath); err != nil {
		return nil, err
	}
	if err := directory.loadSubnets(dirpath, subnetIds); err != nil {
		return nil, err
	}
	dirnames, err := readDirnames(dirpath)
	if err != nil {
		return nil, err
	}
	for _, name := range dirnames {
		path := filepath.Join(dirname, name)
		fi, err := os.Lstat(filepath.Join(topDir, path))
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			continue
		}
		subnetIds := cloneSet(subnetIds)
		if subdir, err := t.readDirectory(topDir, path, subnetIds); err != nil {
			return nil, err
		} else {
			subdir.Name = name
			subdir.parent = directory
			directory.Directories = append(directory.Directories, subdir)
			directory.nameToDirectory[name] = subdir
		}
	}
	return directory, nil
}

func readDirnames(dirname string) ([]string, error) {
	if file, err := os.Open(dirname); err != nil {
		return nil, err
	} else {
		defer file.Close()
		dirnames, err := file.Readdirnames(-1)
		sort.Strings(dirnames)
		return dirnames, err
	}
}

func (directory *Directory) loadMachines(dirname string) error {
	var err error
	directory.Machines, err = loadMachines(
		filepath.Join(dirname, "machines.json"))
	return err
}

func (directory *Directory) loadSubnets(dirname string,
	subnetIds map[string]struct{}) error {
	var err error
	directory.Subnets, err = loadSubnets(filepath.Join(dirname, "subnets.json"))
	if err != nil {
		return err
	}
	for _, subnet := range directory.Subnets {
		if _, ok := subnetIds[subnet.Id]; ok {
			return fmt.Errorf("duplicate subnet ID: %s", subnet.Id)
		} else {
			subnetIds[subnet.Id] = struct{}{}
			directory.subnetIdToSubnet[subnet.Id] = subnet
		}
	}
	return nil
}

func (t *Topology) loadMachines(directory *Directory, dirname string) error {
	if err := directory.loadMachines(dirname); err != nil {
		return err
	}
	for _, machine := range directory.Machines {
		if _, ok := t.machineParents[machine.Hostname]; ok {
			return fmt.Errorf("duplicate machine name: %s", machine.Hostname)
		}
		t.machineParents[machine.Hostname] = directory
	}
	return nil
}
