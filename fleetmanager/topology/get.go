package topology

import (
	"fmt"
)

func (t *Topology) getLocationOfMachine(name string) (string, error) {
	if directory, ok := t.machineParents[name]; !ok {
		return "", fmt.Errorf("unknown machine: %s", name)
	} else {
		return directory.path, nil
	}
}

func (t *Topology) getSubnetsForMachine(name string) ([]*Subnet, error) {
	if directory, ok := t.machineParents[name]; !ok {
		return nil, fmt.Errorf("unknown machine: %s", name)
	} else {
		var subnets []*Subnet
		for ; directory != nil; directory = directory.parent {
			subnets = append(subnets, directory.Subnets...)
		}
		return subnets, nil
	}
}
