package hypervisors

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/url"
	"github.com/Symantec/Dominator/lib/verstr"
)

func (m *Manager) getVMs(doSort bool) []vmInfoType {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	vms := make([]vmInfoType, 0, len(m.vms))
	if doSort {
		ipAddrs := make([]string, 0, len(m.vms))
		for ipAddr := range m.vms {
			ipAddrs = append(ipAddrs, ipAddr)
		}
		verstr.Sort(ipAddrs)
		for _, ipAddr := range ipAddrs {
			vms = append(vms, *m.vms[ipAddr])
		}
	} else {
		for _, vm := range m.vms {
			vms = append(vms, *vm)
		}
	}
	return vms
}

func (m *Manager) listVMsHandler(w http.ResponseWriter,
	req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	topology, err := m.getTopology()
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	parsedQuery := url.ParseQuery(req.URL)
	vms := m.getVMs(true)
	if parsedQuery.OutputType() == url.OutputTypeJson {
		json.WriteWithIndent(writer, "   ", vms)
	}
	if parsedQuery.OutputType() == url.OutputTypeHtml {
		fmt.Fprintf(writer, "<title>List of VMs</title>\n")
		fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
		fmt.Fprintln(writer, "<body>")
		fmt.Fprintln(writer, `<table border="1" style="width:100%">`)
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>IP Addr</th>")
		fmt.Fprintln(writer, "    <th>Name(tag)</th>")
		fmt.Fprintln(writer, "    <th>State</th>")
		fmt.Fprintln(writer, "    <th>RAM</th>")
		fmt.Fprintln(writer, "    <th>CPU</th>")
		fmt.Fprintln(writer, "    <th>Num Volumes</th>")
		fmt.Fprintln(writer, "    <th>Storage</th>")
		fmt.Fprintln(writer, "    <th>Primary Owner</th>")
		fmt.Fprintln(writer, "    <th>Hypervisor</th>")
		fmt.Fprintln(writer, "    <th>Location</th>")
		fmt.Fprintln(writer, "  </tr>")
	}
	for _, vm := range vms {
		switch parsedQuery.OutputType() {
		case url.OutputTypeText:
			fmt.Fprintln(writer, vm.ipAddr)
		case url.OutputTypeHtml:
			var rowStyle []string
			if vm.hypervisor.probeStatus != probeStatusGood {
				rowStyle = append(rowStyle, "color:grey")
			}
			if vm.Uncommitted {
				rowStyle = append(rowStyle, "background-color:yellow")
			} else if topology.CheckIfIpIsReserved(vm.ipAddr) {
				rowStyle = append(rowStyle, "background-color:orange")
			}
			if len(rowStyle) < 1 {
				fmt.Fprintln(writer, "  <tr>")
			} else {
				fmt.Fprintf(writer, "  <tr style=\"%s\">\n",
					strings.Join(rowStyle, ";"))
			}
			fmt.Fprintf(writer,
				"    <td><a href=\"http://%s:%d/showVM?%s\">%s</a></td>\n",
				vm.hypervisor.machine.Hostname, constants.HypervisorPortNumber,
				vm.ipAddr, vm.ipAddr)
			fmt.Fprintf(writer, "    <td>%s</td>\n", vm.Tags["Name"])
			fmt.Fprintf(writer, "    <td>%s</td>\n", vm.State)
			fmt.Fprintf(writer, "    <td>%s</td>\n",
				format.FormatBytes(vm.MemoryInMiB<<20))
			fmt.Fprintf(writer, "    <td>%g</td>\n",
				float64(vm.MilliCPUs)*1e-3)
			fmt.Fprintf(writer, "    <td>%d</td>\n", len(vm.Volumes))
			var storage uint64
			for _, volume := range vm.Volumes {
				storage += volume.Size
			}
			fmt.Fprintf(writer, "    <td>%s</td>\n",
				format.FormatBytes(storage))
			fmt.Fprintf(writer, "    <td>%s</td>\n", vm.OwnerUsers[0])
			fmt.Fprintf(writer,
				"    <td><a href=\"http://%s:%d/\">%s</a></td>\n",
				vm.hypervisor.machine.Hostname, constants.HypervisorPortNumber,
				vm.hypervisor.machine.Hostname)
			location, _ := topology.GetLocationOfMachine(
				vm.hypervisor.machine.Hostname)
			fmt.Fprintf(writer, "    <td>%s</td>\n", location)
			fmt.Fprintf(writer, "  </tr>\n")
		}
	}
	switch parsedQuery.OutputType() {
	case url.OutputTypeHtml:
		fmt.Fprintln(writer, "</table>")
		fmt.Fprintln(writer, "</body>")
	}
}

func (m *Manager) listVMsInLocation(dirname string) ([]net.IP, error) {
	hypervisors, err := m.listHypervisors(dirname, true)
	if err != nil {
		return nil, err
	}
	addresses := make([]net.IP, 0)
	for _, hypervisor := range hypervisors {
		hypervisor.mutex.RLock()
		for _, vm := range hypervisor.vms {
			addresses = append(addresses, vm.Address.IpAddress)
		}
		hypervisor.mutex.RUnlock()
	}
	return addresses, nil
}
