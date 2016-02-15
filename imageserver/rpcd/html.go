package rpcd

import (
	"fmt"
	"io"
)

func (hw *htmlWriter) writeHtml(writer io.Writer) {
	fmt.Fprintf(writer, "Replication clients: %d<br>\n",
		hw.getNumReplicationClients())
}

func (hw *htmlWriter) getNumReplicationClients() uint {
	hw.numReplicationClientsLock.RLock()
	defer hw.numReplicationClientsLock.RUnlock()
	return hw.numReplicationClients
}
