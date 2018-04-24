package manager

import (
	"fmt"
	"io"
	"strings"

	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/meminfo"
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
	numAddresses := len(m.addressPool)
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
	fmt.Fprintf(writer,
		"Number of subnets: <a href=\"listSubnets\">%d</a><br>\n", numSubnets)
	fmt.Fprintf(writer, "Volume directories: %s<br>\n",
		strings.Join(m.volumeDirectories, " "))
}

func writeCountLinks(writer io.Writer, text, path string, count uint) {
	fmt.Fprintf(writer,
		"%s: <a href=\"%s\">%d</a> (<a href=\"%s&output=text\">text</a>)<br>\n",
		text, path, count, path)
}
