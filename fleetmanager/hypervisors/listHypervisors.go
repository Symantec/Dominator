package hypervisors

import (
	"bufio"
	"fmt"
	"net/http"
	"sort"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/url"
	proto "github.com/Symantec/Dominator/proto/fleetmanager"
)

const (
	showOK = iota
	showConnected
	showAll
)

type hypervisorList []*hypervisorType

func (h *hypervisorType) getHealthStatus() string {
	healthStatus := h.probeStatus.String()
	if h.probeStatus == probeStatusConnected {
		if h.healthStatus != "" {
			healthStatus = h.healthStatus
		}
	}
	return healthStatus
}

func (h *hypervisorType) getNumVMs() uint {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return uint(len(h.vms))
}

func (m *Manager) listHypervisors(topologyDir string, showFilter int,
	subnetId string) (hypervisorList, error) {
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
		switch showFilter {
		case showOK:
			if hypervisor.probeStatus == probeStatusConnected &&
				(hypervisor.healthStatus == "" ||
					hypervisor.healthStatus == "healthy") {
				hypervisors = append(hypervisors, hypervisor)
			}
		case showConnected:
			if hypervisor.probeStatus == probeStatusConnected {
				hypervisors = append(hypervisors, hypervisor)
			}
		case showAll:
			hypervisors = append(hypervisors, hypervisor)
		}
	}
	return hypervisors, nil
}

func (m *Manager) listHypervisorsHandler(w http.ResponseWriter,
	req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	_, err := m.getTopology()
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	parsedQuery := url.ParseQuery(req.URL)
	showFilter := showAll
	switch parsedQuery.Table["state"] {
	case "connected":
		showFilter = showConnected
	case "OK":
		showFilter = showOK
	}
	hypervisors, err := m.listHypervisors("", showFilter, "")
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	sort.Sort(hypervisors)
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
	writer.WriteString(commonStyleSheet)
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
	lastRowHighlighted := true
	for _, hypervisor := range hypervisors {
		machine := hypervisor.machine
		if lastRowHighlighted {
			lastRowHighlighted = false
			fmt.Fprintf(writer, "  <tr>\n")
		} else {
			lastRowHighlighted = true
			fmt.Fprintf(writer, "  <tr style=\"%s\">\n",
				rowStyles[rowStyleHighlight].html)
		}
		fmt.Fprintf(writer,
			"    <td><a href=\"showHypervisor?%s\">%s</a></td>\n",
			machine.Hostname, machine.Hostname)
		fmt.Fprintf(writer, "    <td><a href=\"http://%s:%d/\">%s</a></td>\n",
			machine.Hostname, constants.HypervisorPortNumber,
			hypervisor.getHealthStatus())
		fmt.Fprintf(writer, "    <td>%s</td>\n", machine.HostIpAddress)
		fmt.Fprintf(writer, "    <td>%s</td>\n", machine.HostMacAddress)
		fmt.Fprintf(writer, "    <td>%s</td>\n", hypervisor.location)
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
	hypervisors, err := m.listHypervisors(request.Location, showConnected,
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

func (list hypervisorList) Len() int {
	return len(list)
}

func (list hypervisorList) Less(i, j int) bool {
	if list[i].location < list[j].location {
		return true
	} else if list[i].location > list[j].location {
		return false
	} else {
		return list[i].machine.Hostname < list[j].machine.Hostname
	}
}

func (list hypervisorList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}
