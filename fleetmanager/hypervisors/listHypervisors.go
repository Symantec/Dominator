package hypervisors

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/url"
	proto "github.com/Symantec/Dominator/proto/fleetmanager"
)

func (h *hypervisorType) getNumVMs() uint {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return uint(len(h.vms))
}

func (m *Manager) listHypervisors(topologyDir string, connectedOnly bool,
	subnetId string) (
	[]*hypervisorType, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	machines, err := m.topology.ListMachines(topologyDir)
	if err != nil {
		return nil, err
	}
	hypervisors := make([]*hypervisorType, 0, len(machines))
	for _, machine := range machines {
		if subnetId != "" {
			hasSubnet, _ := m.topology.CheckIfMachineHasSubnet(
				machine.Hostname, subnetId)
			if !hasSubnet {
				continue
			}
		}
		hypervisor := m.hypervisors[machine.Hostname]
		if !connectedOnly || hypervisor.probeStatus == probeStatusGood {
			hypervisors = append(hypervisors, hypervisor)
		}
	}
	return hypervisors, nil
}

func (m *Manager) listHypervisorsHandler(w http.ResponseWriter,
	req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	topology, err := m.getTopology()
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	parsedQuery := url.ParseQuery(req.URL)
	matchConnectedOnly := false
	if parsedQuery.Table["state"] == "connected" {
		matchConnectedOnly = true
	}
	hypervisors, err := m.listHypervisors("", matchConnectedOnly, "")
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	if parsedQuery.OutputType() == url.OutputTypeText {
		for _, hypervisor := range hypervisors {
			fmt.Fprintln(writer, hypervisor.machine.Hostname)
		}
		return
	}
	if parsedQuery.OutputType() == url.OutputTypeJson {
		json.WriteWithIndent(writer, "    ", hypervisors)
		return
	}
	fmt.Fprintf(writer, "<title>List of hypervisors</title>\n")
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, `<table border="1" style="width:100%">`)
	fmt.Fprintln(writer, "  <tr>")
	fmt.Fprintln(writer, "    <th>Name</th>")
	fmt.Fprintln(writer, "    <th>Status</th>")
	fmt.Fprintln(writer, "    <th>IP Addr</th>")
	fmt.Fprintln(writer, "    <th>MAC Addr</th>")
	fmt.Fprintln(writer, "    <th>Location</th>")
	fmt.Fprintln(writer, "    <th>NumVMs</th>")
	fmt.Fprintln(writer, "  </tr>")
	for _, hypervisor := range hypervisors {
		if matchConnectedOnly &&
			hypervisor.probeStatus != probeStatusGood {
			continue
		}
		machine := hypervisor.machine
		fmt.Fprintf(writer, "  <tr>\n")
		fmt.Fprintf(writer, "    <td><a href=\"http://%s:%d/\">%s</a></td>\n",
			machine.Hostname, constants.HypervisorPortNumber, machine.Hostname)
		fmt.Fprintf(writer, "    <td>%s</td>\n", hypervisor.probeStatus)
		fmt.Fprintf(writer, "    <td>%s</td>\n", machine.HostIpAddress)
		fmt.Fprintf(writer, "    <td>%s</td>\n", machine.HostMacAddress)
		location, _ := topology.GetLocationOfMachine(machine.Hostname)
		fmt.Fprintf(writer, "    <td>%s</td>\n", location)
		fmt.Fprintf(writer,
			"    <td><a href=\"http://%s:%d/listVMs\">%d</a></td>\n",
			machine.Hostname, constants.HypervisorPortNumber,
			hypervisor.getNumVMs())
		fmt.Fprintf(writer, "  </tr>\n")
	}
	fmt.Fprintln(writer, "</table>")
	fmt.Fprintln(writer, "</body>")
}

func (m *Manager) listHypervisorsInLocation(
	request proto.ListHypervisorsInLocationRequest) ([]string, error) {
	hypervisors, err := m.listHypervisors(request.Location, true,
		request.SubnetId)
	if err != nil {
		return nil, err
	}
	addresses := make([]string, 0, len(hypervisors))
	for _, hypervisor := range hypervisors {
		addresses = append(addresses,
			fmt.Sprintf("%s:%d",
				hypervisor.machine.Hostname, constants.HypervisorPortNumber))
	}
	return addresses, nil
}
