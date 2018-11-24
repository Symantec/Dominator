package httpd

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/url"
)

func (s state) showVMHandler(w http.ResponseWriter, req *http.Request) {
	parsedQuery := url.ParseQuery(req.URL)
	if len(parsedQuery.Flags) != 1 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var ipAddr string
	for name := range parsedQuery.Flags {
		ipAddr = name
	}
	netIpAddr := net.ParseIP(ipAddr)
	vm, err := s.manager.GetVmInfo(netIpAddr)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	if parsedQuery.OutputType() == url.OutputTypeJson {
		json.WriteWithIndent(writer, "    ", vm)
	} else {
		var storage uint64
		volumeSizes := make([]string, 0, len(vm.Volumes))
		for _, volume := range vm.Volumes {
			storage += volume.Size
			volumeSizes = append(volumeSizes, format.FormatBytes(volume.Size))
		}
		var tagNames []string
		for name := range vm.Tags {
			tagNames = append(tagNames, name)
		}
		sort.Strings(tagNames)
		fmt.Fprintf(writer, "<title>Information for VM %s</title>\n", ipAddr)
		fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
		fmt.Fprintln(writer, "<body>")
		fmt.Fprintln(writer, `<table border="0">`)
		if len(vm.Address.IpAddress) < 1 {
			writeString(writer, "IP Address", ipAddr+" (externally allocated)")
		} else if vm.Uncommitted {
			writeString(writer, "IP Address", ipAddr+" (uncommitted)")
		} else {
			writeString(writer, "IP Address", ipAddr)
		}
		if vm.Hostname != "" {
			writeString(writer, "Hostname", vm.Hostname)
		}
		writeString(writer, "MAC Address", vm.Address.MacAddress)
		if vm.ImageName != "" {
			image := fmt.Sprintf("<a href=\"http://%s/showImage?%s\">%s</a>",
				s.manager.GetImageServerAddress(), vm.ImageName, vm.ImageName)
			writeString(writer, "Boot image", image)
		} else if vm.ImageURL != "" {
			writeString(writer, "Boot image URL", vm.ImageURL)
		} else {
			writeString(writer, "Boot image", "was streamed in")
		}
		writeString(writer, "State", vm.State.String())
		writeString(writer, "RAM", format.FormatBytes(vm.MemoryInMiB<<20))
		writeFloat(writer, "CPU", float64(vm.MilliCPUs)*1e-3)
		writeStrings(writer, "Volume sizes", volumeSizes)
		writeString(writer, "Total storage", format.FormatBytes(storage))
		writeStrings(writer, "Owner users", vm.OwnerGroups)
		writeStrings(writer, "Owner users", vm.OwnerUsers)
		writeBool(writer, "Spread volumes", vm.SpreadVolumes)
		writeString(writer, "Latest boot",
			fmt.Sprintf("<a href=\"showVmBootLog?%s\">log</a>", ipAddr))
		if ok, _ := s.manager.CheckVmHasHealthAgent(netIpAddr); ok {
			writeString(writer, "Health Agent",
				fmt.Sprintf("<a href=\"http://%s:6910/\">detected</a>",
					ipAddr))
		}
		fmt.Fprintln(writer, "</table>")
		fmt.Fprintln(writer, "Tags:<br>")
		fmt.Fprintln(writer, `<table border="1">`)
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>Name</th>")
		fmt.Fprintln(writer, "    <th>Value</th>")
		fmt.Fprintln(writer, "  </tr>")
		for _, name := range tagNames {
			writeString(writer, name, vm.Tags[name])
		}
		fmt.Fprintln(writer, "</table><br>")
		fmt.Fprintf(writer,
			"<a href=\"showVM?%s&output=json\">VM info:</a><br>\n",
			vm.Address.IpAddress)
		fmt.Fprintln(writer, `<pre style="background-color: #eee; border: 1px solid #999; display: block; float: left;">`)
		json.WriteWithIndent(writer, "    ", vm)
		fmt.Fprintln(writer, `</pre><p style="clear: both;">`)
		fmt.Fprintln(writer, "</body>")
	}
}

func writeBool(writer io.Writer, name string, value bool) {
	fmt.Fprintf(writer, "  <tr><td>%s</td><td>%t</td></tr>\n", name, value)
}

func writeInt(writer io.Writer, name string, value int) {
	fmt.Fprintf(writer, "  <tr><td>%s</td><td>%d</td></tr>\n", name, value)
}

func writeFloat(writer io.Writer, name string, value float64) {
	fmt.Fprintf(writer, "  <tr><td>%s</td><td>%g</td></tr>\n", name, value)
}

func writeString(writer io.Writer, name, value string) {
	fmt.Fprintf(writer, "  <tr><td>%s</td><td>%s</td></tr>\n", name, value)
}

func writeStrings(writer io.Writer, name string, value []string) {
	if len(value) < 1 {
		return
	}
	fmt.Fprintf(writer, "  <tr><td>%s</td><td>%s</td></tr>\n",
		name, strings.Join(value, ", "))
}

func writeUint64(writer io.Writer, name string, value uint64) {
	fmt.Fprintf(writer, "  <tr><td>%s</td><td>%d</td></tr>\n", name, value)
}
