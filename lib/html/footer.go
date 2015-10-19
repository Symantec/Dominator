package html

import (
	"fmt"
	"io"
	"time"
)

func writeFooter(writer io.Writer) {
	fmt.Fprintf(writer, "Page generated at: %s<br>\n",
		time.Now().Format(timeFormat))
}
