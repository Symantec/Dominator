package hypervisors

import (
	"fmt"
	"io"
)

func (m *Manager) writeHtml(writer io.Writer) {
	t, err := m.getTopology()
	if err != nil {
		fmt.Fprintln(writer, err, "<br>")
		return
	}
	if *manageHypervisors {
		fmt.Fprintln(writer,
			`Hypervisors <font color="green">are</font> being managed by this instance<br>`)
	} else {
		fmt.Fprintln(writer,
			`<font color="grey">Hypervisors are not being managed by this instance</font><br>`)
	}
	numMachines := t.GetNumMachines()
	var numConnected uint
	m.mutex.RLock()
	for _, hypervisor := range m.hypervisors {
		if hypervisor.probeStatus == probeStatusGood {
			numConnected++
		}
	}
	numVMs := uint(len(m.vms))
	m.mutex.RUnlock()
	writeCountLinksHT(writer, "Number of hypervisors known",
		"listHypervisors?state=", numMachines)
	writeCountLinksHT(writer, "Number of hypervisors connected",
		"listHypervisors?state=connected", numConnected)
	writeCountLinksHTJ(writer, "Number of VMs known",
		"listVMs?", numVMs)
	fmt.Fprintln(writer, `Hypervisor <a href="listLocations">locations</a><br>`)
}

func writeCountLinksHT(writer io.Writer, text, path string, count uint) {
	fmt.Fprintf(writer,
		"%s: <a href=\"%s\">%d</a> (<a href=\"%s&output=text\">text</a>)<br>\n",
		text, path, count, path)
}

func writeCountLinksHTJ(writer io.Writer, text, path string, count uint) {
	fmt.Fprintf(writer,
		"%s: <a href=\"%s\">%d</a> (<a href=\"%s&output=text\">text</a>, <a href=\"%s&output=json\">JSON</a>)<br>\n",
		text, path, count, path, path)
}
