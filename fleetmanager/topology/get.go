package topology

import (
	"fmt"
)

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
