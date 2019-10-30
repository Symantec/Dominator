package httpd

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/url"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (s state) listVMsHandler(w http.ResponseWriter, req *http.Request) {
	parsedQuery := url.ParseQuery(req.URL)
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	ipAddrs := s.manager.ListVMs(nil, true)
	matchState := parsedQuery.Table["state"]
	if parsedQuery.OutputType() == url.OutputTypeText && matchState == "" {
		for _, ipAddr := range ipAddrs {
			fmt.Fprintln(writer, ipAddr)
		}
		return
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
		fmt.Fprintln(writer, "    <th>MAC Addr</th>")
		fmt.Fprintln(writer, "    <th>Name(tag)</th>")
		fmt.Fprintln(writer, "    <th>State</th>")
		fmt.Fprintln(writer, "    <th>RAM</th>")
		fmt.Fprintln(writer, "    <th>CPU</th>")
		fmt.Fprintln(writer, "    <th>Num Volumes</th>")
		fmt.Fprintln(writer, "    <th>Storage</th>")
		fmt.Fprintln(writer, "    <th>Primary Owner</th>")
		fmt.Fprintln(writer, "  </tr>")
	}
	for _, ipAddr := range ipAddrs {
		vm, err := s.manager.GetVmInfo(net.ParseIP(ipAddr))
		if err != nil {
			continue
		}
		if matchState != "" && matchState != vm.State.String() {
			continue
		}
		switch parsedQuery.OutputType() {
		case url.OutputTypeText:
			fmt.Fprintln(writer, ipAddr)
		case url.OutputTypeHtml:
			if vm.Uncommitted {
				fmt.Fprintln(writer, "  <tr style=\"background-color:yellow\">")
			} else {
				fmt.Fprintln(writer, "  <tr>")
			}
			fmt.Fprintf(writer, "    <td><a href=\"showVM?%s\">%s</a></td>\n",
				ipAddr, ipAddr)
			fmt.Fprintf(writer, "    <td>%s</td>\n", vm.Address.MacAddress)
			fmt.Fprintf(writer, "    <td>%s</td>\n", vm.Tags["Name"])
			fmt.Fprintf(writer, "    <td>%s</td>\n", vm.State)
			fmt.Fprintf(writer, "    <td>%s</td>\n",
				format.FormatBytes(vm.MemoryInMiB<<20))
			fmt.Fprintf(writer, "    <td>%g</td>\n",
				float64(vm.MilliCPUs)*1e-3)
			writeNumVolumesTableEntry(writer, vm)
			writeStorageTotalTableEntry(writer, vm)
			fmt.Fprintf(writer, "    <td>%s</td>\n", vm.OwnerUsers[0])
			fmt.Fprintf(writer, "  </tr>\n")
		}
	}
	switch parsedQuery.OutputType() {
	case url.OutputTypeHtml:
		fmt.Fprintln(writer, "</table>")
		fmt.Fprintln(writer, "</body>")
	}
}

func writeNumVolumesTableEntry(writer io.Writer, vm proto.VmInfo) {
	var comment string
	for _, volume := range vm.Volumes {
		if comment == "" && volume.Format != proto.VolumeFormatRaw {
			comment = `<font style="color:grey;font-size:12px"> (!RAW)</font>`
		}
	}
	fmt.Fprintf(writer, "    <td>%d%s</td>\n", len(vm.Volumes), comment)
}

func writeStorageTotalTableEntry(writer io.Writer, vm proto.VmInfo) {
	var storage uint64
	for _, volume := range vm.Volumes {
		storage += volume.Size
	}
	fmt.Fprintf(writer, "    <td>%s</td>\n", format.FormatBytes(storage))
}
