package topology

import (
	"fmt"
)

func (t *Topology) checkIfMachineHasSubnet(name, subnetId string) (
	bool, error) {
	if directory, ok := t.machineParents[name]; !ok {
		return false, fmt.Errorf("unknown machine: %s", name)
	} else {
		// TODO(rgooch): It would be faster to have a single map for all the
		//               subnets down to the directory, but it would consume
		//               more memory. Revisit this if needed.
		for ; directory != nil; directory = directory.parent {
			if _, ok := directory.subnetIdToSubnet[subnetId]; ok {
				return true, nil
			}
		}
		return false, nil
	}
}
