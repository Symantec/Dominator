package rpcd

import (
	"fmt"
	"io"
)

func (hw *htmlWriter) writeHtml(writer io.Writer) {
	fmt.Fprintf(writer, "GetObjects() RPC slots: %d out of %d<br>\n",
		len(hw.getSemaphore), cap(hw.getSemaphore))
}
