package html

import (
	"fmt"
	"io"
	"time"

	"github.com/Symantec/Dominator/lib/format"
)

func writeFooter(writer io.Writer) {
	fmt.Fprintf(writer, "Page generated at: %s<br>\n",
		time.Now().Format(format.TimeFormatSeconds))
}
