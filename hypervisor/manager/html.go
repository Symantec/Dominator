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
	fmt.Fprintf(writer,
		"Number of VMs <a href=\"listVMs?output=text\">known</a>: <a href=\"listVMs\">%d</a><br>\n",
		numRunning+numStopped)
	fmt.Fprintf(writer,
		"Number of VMs <a href=\"listVMs?state=running&output=text\">running</a>: <a href=\"listVMs?state=running\">%d</a><br>\n",
		numRunning)
	fmt.Fprintf(writer,
		"Number of VMs <a href=\"listVMs?state=stopped&output=text\">stopped</a>: <a href=\"listVMs?state=stopped\">%d</a><br>\n",
		numStopped)
	fmt.Fprintln(writer, "<br>")
	m.mutex.RLock()
	memUnallocated := m.getUnallocatedMemoryInMiBWithLock()
	numSubnets := len(m.subnets)
	numAddresses := len(m.addressPool)
	m.mutex.RUnlock()
	fmt.Fprintf(writer, "Available addresses: %d<br>\n", numAddresses)
	fmt.Fprintf(writer, "Available CPU: %g<br>\n",
		float64(m.getAvailableMilliCPU())*1e-3)
	if memInfo, err := meminfo.GetMemInfo(); err != nil {
		fmt.Fprintf(writer, "Error getting available RAM: %s<br>\n", err)
	} else {
		fmt.Fprintf(writer, "Available RAM: real: %s, unallocated: %s<br>\n",
			format.FormatBytes(memInfo.Available),
			format.FormatBytes(memUnallocated<<20))
	}
	fmt.Fprintf(writer, "Number of subnets: %d<br>\n", numSubnets)
	fmt.Fprintf(writer, "Volume directories: %s<br>\n",
		strings.Join(m.volumeDirectories, " "))
}
