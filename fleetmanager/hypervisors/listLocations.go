package hypervisors

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/Symantec/Dominator/fleetmanager/topology"
)

func (m *Manager) listLocations(dirname string) ([]string, error) {
	topo, err := m.getTopology()
	if err != nil {
		return nil, err
	}
	directory, err := topo.FindDirectory(dirname)
	if err != nil {
		return nil, err
	}
	var locations []string
	directory.Walk(func(directory *topology.Directory) error {
		for _, machine := range directory.Machines {
			hypervisor, err := m.getLockedHypervisor(machine.Hostname, false)
			if err != nil {
				continue
			}
			if hypervisor.probeStatus == probeStatusGood {
				locations = append(locations, directory.GetPath())
				hypervisor.mutex.RUnlock()
				return nil
			}
			hypervisor.mutex.RUnlock()
		}
		return nil
	})
	return locations, nil
}

func (m *Manager) listLocationsHandler(w http.ResponseWriter,
	req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	if locations, err := m.listLocations(""); err != nil {
		fmt.Fprintln(writer, err)
	} else {
		for _, location := range locations {
			fmt.Fprintln(writer, location)
		}
	}
}
