package hypervisors

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/url"
	"github.com/Symantec/Dominator/lib/verstr"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

const commonStyleSheet string = `<style>
table, th, td {
border-collapse: collapse;
}
</style>
`

const (
	rowStyleProbeBad = iota
	rowStyleOff
	rowStyleHealthMarginal
	rowStyleHealthAtRisk
	rowStyleHighlight
	rowStyleReservedIP
	rowStyleUncommittedIP
)

type rowStyleType struct {
	html        string
	highlighted bool
}

var (
	rowStyles = map[uint]rowStyleType{
		rowStyleProbeBad:       {"color:red", false},
		rowStyleOff:            {"color:#ff8080", false},
		rowStyleHealthMarginal: {"color:#800000", false},
		rowStyleHealthAtRisk:   {"color:#c00000", false},
		rowStyleHighlight:      {"background-color:#fafafa", true},
		rowStyleReservedIP:     {"background-color:orange", true},
		rowStyleUncommittedIP:  {"background-color:yellow", true},
	}
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
	primaryOwnerFilter := parsedQuery.Table["primaryOwner"]
	if parsedQuery.OutputType() == url.OutputTypeHtml {
		fmt.Fprintf(writer, "<title>List of VMs</title>\n")
		writer.WriteString(commonStyleSheet)
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
	lastRowHighlighted := true
	primaryOwnersMap := make(map[string]struct{})
	for _, vm := range vms {
		if primaryOwnerFilter != "" {
			if vm.OwnerUsers[0] != primaryOwnerFilter {
				primaryOwnersMap[vm.OwnerUsers[0]] = struct{}{}
				continue
			}
		} else {
			primaryOwnersMap[vm.OwnerUsers[0]] = struct{}{}
		}
		switch parsedQuery.OutputType() {
		case url.OutputTypeText:
			fmt.Fprintln(writer, vm.ipAddr)
		case url.OutputTypeHtml:
			var rowStyle []rowStyleType
			if vm.hypervisor.probeStatus == probeStatusOff {
				rowStyle = append(rowStyle, rowStyles[rowStyleOff])
			} else if vm.hypervisor.probeStatus != probeStatusConnected {
				rowStyle = append(rowStyle, rowStyles[rowStyleProbeBad])
			} else if vm.hypervisor.healthStatus == "at risk" {
				rowStyle = append(rowStyle, rowStyles[rowStyleHealthAtRisk])
			} else if vm.hypervisor.healthStatus == "marginal" {
				rowStyle = append(rowStyle, rowStyles[rowStyleHealthMarginal])
			}
			if vm.Uncommitted {
				rowStyle = append(rowStyle, rowStyles[rowStyleUncommittedIP])
			} else if topology.CheckIfIpIsReserved(vm.ipAddr) {
				rowStyle = append(rowStyle, rowStyles[rowStyleReservedIP])
			}
			styles := make([]string, 0, len(rowStyle))
			highlighted := false
			for _, style := range rowStyle {
				styles = append(styles, style.html)
				if style.highlighted {
					highlighted = true
				}
			}
			if !highlighted && !lastRowHighlighted {
				styles = append(styles, rowStyles[rowStyleHighlight].html)
				highlighted = true
			}
			lastRowHighlighted = highlighted
			if len(styles) < 1 {
				fmt.Fprintln(writer, "  <tr>")
			} else {
				fmt.Fprintf(writer, "  <tr style=\"%s\">\n",
					strings.Join(styles, ";"))
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
			vm.writeNumVolumesTableEntry(writer)
			vm.writeStorageTotalTableEntry(writer)
			fmt.Fprintf(writer, "    <td>%s</td>\n", vm.OwnerUsers[0])
			fmt.Fprintf(writer,
				"    <td><a href=\"http://%s:%d/\">%s</a></td>\n",
				vm.hypervisor.machine.Hostname, constants.HypervisorPortNumber,
				vm.hypervisor.machine.Hostname)
			fmt.Fprintf(writer, "    <td>%s</td>\n", vm.hypervisor.location)
			fmt.Fprintf(writer, "  </tr>\n")
		}
	}
	switch parsedQuery.OutputType() {
	case url.OutputTypeHtml:
		fmt.Fprintln(writer, "</table>")
		fmt.Fprintln(writer, "</body>")
		primaryOwners := make([]string, 0, len(primaryOwnersMap))
		for primaryOwner := range primaryOwnersMap {
			primaryOwners = append(primaryOwners, primaryOwner)
		}
		sort.Strings(primaryOwners)
		fmt.Fprintln(writer, "Filter by primary owner:<br>")
		for _, primaryOwner := range primaryOwners {
			fmt.Fprintf(writer,
				"<a href=\"listVMs?primaryOwner=%s\">%s</a><br>\n",
				primaryOwner, primaryOwner)
		}
	}
}

func (m *Manager) listVMsInLocation(dirname string) ([]net.IP, error) {
	hypervisors, err := m.listHypervisors(dirname, showAll, "")
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

func (vm *vmInfoType) writeNumVolumesTableEntry(writer io.Writer) {
	var comment string
	for _, volume := range vm.Volumes {
		if comment == "" && volume.Format != proto.VolumeFormatRaw {
			comment = `<font style="color:grey;font-size:12px"> (!RAW)</font>`
		}
	}
	fmt.Fprintf(writer, "    <td>%d%s</td>\n", len(vm.Volumes), comment)
}

func (vm *vmInfoType) writeStorageTotalTableEntry(writer io.Writer) {
	var storage uint64
	for _, volume := range vm.Volumes {
		storage += volume.Size
	}
	fmt.Fprintf(writer, "    <td>%s</td>\n", format.FormatBytes(storage))
}
