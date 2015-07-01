package scanner

import (
	"fmt"
	"github.com/Symantec/Dominator/sub/fsrateio"
	"io"
	"time"
)

func (fsh *FileSystemHistory) writeHtml(writer io.Writer) {
	fmt.Fprintf(writer, "Scan count: %d<br>\n", fsh.scanCount)
	fmt.Fprintf(writer, "Generation count: %d<br>\n", fsh.generationCount)
	if fsh.scanCount > 0 {
		fmt.Fprintf(writer, "Last scan completed: %s<br>\n", fsh.timeOfLastScan)
		fmt.Fprintf(writer, "Duration of last scan: %s<br>\n",
			fsh.durationOfLastScan)
		fsh.fileSystem.WriteHtml(writer)
		tmp := fsrateio.FormatBytes(uint64(float64(
			fsh.fileSystem.TotalDataBytes) / fsh.durationOfLastScan.Seconds()))
		fmt.Fprintf(writer, "Scan rate: %s/s<br>\n", tmp)
	}
	fmt.Fprintf(writer, "Duration of current scan: %s<br>\n",
		time.Since(fsh.timeOfLastScan))
	if fsh.generationCount > 0 {
		fmt.Fprintf(writer, "Last change: %s<br>\n", fsh.timeOfLastChange)
	}
}

func (fs *FileSystem) writeHtml(writer io.Writer) {
	fmt.Fprintf(writer, "Scanned: %s<br>\n",
		fsrateio.FormatBytes(fs.TotalDataBytes))
}
