package rpcd

import (
	"fmt"
	"io"
)

func (hw *HtmlWriter) writeHtml(writer io.Writer) {
	fmt.Fprintf(writer, "Image of last successful update: \"%s\"<br>\n",
		*hw.lastSuccessfulImageName)
}
