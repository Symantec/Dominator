package hypervisors

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/url"
)

func (m *Manager) listHypervisors(topologyDir string) (
	[]*hypervisorType, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	machines, err := m.topology.ListMachines(topologyDir)
	if err != nil {
		return nil, err
	}
	hypervisors := make([]*hypervisorType, 0, len(machines))
	for _, machine := range machines {
		hypervisors = append(hypervisors, m.hypervisors[machine.Hostname])
	}
	return hypervisors, nil
}

func (m *Manager) listHypervisorsHandler(w http.ResponseWriter,
	req *http.Request) {
	parsedQuery := url.ParseQuery(req.URL)
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	matchConnectedOnly := false
	if parsedQuery.Table["state"] == "connected" {
		matchConnectedOnly = true
	}
	hypervisors, err := m.listHypervisors("")
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	if parsedQuery.OutputType() == url.OutputTypeText {
		for _, hypervisor := range hypervisors {
			if matchConnectedOnly &&
				hypervisor.probeStatus != probeStatusGood {
				continue
			}
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
	fmt.Fprintln(writer, "  </tr>")
	for _, hypervisor := range hypervisors {
		if matchConnectedOnly &&
			hypervisor.probeStatus != probeStatusGood {
			continue
		}
		machine := hypervisor.machine
		fmt.Fprintf(writer, "  <tr>\n")
		fmt.Fprintf(writer, "    <td><a href=\"http://%s:6976/\">%s</a></td>\n",
			machine.Hostname, machine.Hostname)
		fmt.Fprintf(writer, "    <td>%s</td>\n", hypervisor.probeStatus)
		fmt.Fprintf(writer, "    <td>%s</td>\n", machine.HostIpAddress)
		fmt.Fprintf(writer, "    <td>%s</td>\n", machine.HostMacAddress)
		fmt.Fprintf(writer, "  </tr>\n")
	}
	fmt.Fprintln(writer, "</table>")
	fmt.Fprintln(writer, "</body>")
}
