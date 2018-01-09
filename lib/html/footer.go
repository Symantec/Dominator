package html

import (
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/Symantec/Dominator/lib/format"
)

func writeFooter(writer io.Writer) {
	fmt.Fprintf(writer, "Page generated at: %s with %s<br>\n",
		time.Now().Format(format.TimeFormatSeconds),
		runtime.Version())
}
