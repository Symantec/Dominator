package html

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"io"
	"time"
)

func writeFooter(writer io.Writer) {
	fmt.Fprintf(writer, "Page generated at: %s<br>\n",
		time.Now().Format(format.TimeFormatSeconds))
}
