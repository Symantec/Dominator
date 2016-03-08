package filegen

import (
	"fmt"
	"io"
)

func (m *Manager) writeHtml(writer io.Writer) {
	fmt.Fprintf(writer,
		"Number of generated files: <a href=\"listGenerators\">%d</a><br>\n",
		len(m.fileGenerators))
}
