package manager

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/meminfo"
)

func (m *Manager) writeHtml(writer io.Writer) {
	numRunning, numStopped := m.getNumVMs()
	writeCountLinks(writer, "Number of VMs known", "listVMs?state=",
		numRunning+numStopped)
	writeCountLinks(writer, "Number of VMs running", "listVMs?state=running",
		numRunning)
	writeCountLinks(writer, "Number of VMs stopped", "listVMs?state=stopped",
		numStopped)
	fmt.Fprintln(writer, "<br>")
	m.mutex.RLock()
	memUnallocated := m.getUnallocatedMemoryInMiBWithLock()
	numSubnets := len(m.subnets)
	numAddresses := len(m.addressPool.Free)
	ownerGroups := make([]string, 0, len(m.ownerGroups))
	for group := range m.ownerGroups {
		ownerGroups = append(ownerGroups, group)
	}
	ownerUsers := make([]string, 0, len(m.ownerUsers))
	for user := range m.ownerUsers {
		ownerUsers = append(ownerUsers, user)
	}
	m.mutex.RUnlock()
	fmt.Fprintf(writer,
		"Available addresses: <a href=\"listAvailableAddresses\">%d</a><br>\n",
		numAddresses)
	fmt.Fprintf(writer, "Available CPU: %g<br>\n",
		float64(m.getAvailableMilliCPU())*1e-3)
	if memInfo, err := meminfo.GetMemInfo(); err != nil {
		fmt.Fprintf(writer, "Error getting available RAM: %s<br>\n", err)
	} else {
		fmt.Fprintf(writer, "Available RAM: real: %s, unallocated: %s<br>\n",
			format.FormatBytes(memInfo.Available),
			format.FormatBytes(memUnallocated<<20))
	}
	sort.Strings(ownerGroups)
	sort.Strings(ownerUsers)
	if len(ownerGroups) > 0 {
		fmt.Fprintf(writer, "Owner groups: %s<br>\n",
			strings.Join(ownerGroups, " "))
	}
	if len(ownerUsers) > 0 {
		fmt.Fprintf(writer, "Owner users: %s<br>\n",
			strings.Join(ownerUsers, " "))
	}
	if m.serialNumber != "" {
		fmt.Fprintf(writer, "Serial number: \"%s\"<br>\n", m.serialNumber)
	}
	fmt.Fprintf(writer,
		"Number of subnets: <a href=\"listSubnets\">%d</a><br>\n", numSubnets)
	fmt.Fprintf(writer, "Volume directories: %s<br>\n",
		strings.Join(m.volumeDirectories, " "))
	if m.objectCache == nil {
		fmt.Fprintln(writer, "No object cache<br>")
	} else {
		m.objectCache.WriteHtml(writer)
	}
}

func writeCountLinks(writer io.Writer, text, path string, count uint) {
	fmt.Fprintf(writer,
		"%s: <a href=\"%s\">%d</a> (<a href=\"%s&output=text\">text</a>)<br>\n",
		text, path, count, path)
}
