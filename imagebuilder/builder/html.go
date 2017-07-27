package builder

import (
	"fmt"
	"io"
	"sort"
)

func (b *Builder) writeHtml(writer io.Writer) {
	currentBuilds := make([]string, 0)
	goodBuilds := make(map[string]buildResultType)
	failedBuilds := make(map[string]buildResultType)
	b.buildResultsLock.RLock()
	for name := range b.currentBuildLogs {
		currentBuilds = append(currentBuilds, name)
	}
	for name, result := range b.lastBuildResults {
		if result.error == nil {
			goodBuilds[name] = result
		} else {
			failedBuilds[name] = result
		}
	}
	b.buildResultsLock.RUnlock()
	if len(currentBuilds) > 0 {
		fmt.Fprintln(writer, "Current image builds:<br>")
		fmt.Fprintln(writer, `<table border="1">`)
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>Image Stream</th>")
		fmt.Fprintln(writer, "    <th>Build log</th>")
		fmt.Fprintln(writer, "  </tr>")
		for _, streamName := range currentBuilds {
			fmt.Fprintf(writer, "  <tr>\n")
			fmt.Fprintf(writer, "    <td>%s</td>\n", streamName)
			fmt.Fprintf(writer,
				"    <td><a href=\"showCurrentBuildLog?%s#bottom\">log</a></td>\n",
				streamName)
			fmt.Fprintf(writer, "  </tr>\n")
		}
		fmt.Fprintln(writer, "</table><br>")
	}
	if len(failedBuilds) > 0 {
		streamNames := make([]string, 0, len(failedBuilds))
		for streamName := range failedBuilds {
			streamNames = append(streamNames, streamName)
		}
		sort.Strings(streamNames)
		fmt.Fprintln(writer, "Failed image builds:<br>")
		fmt.Fprintln(writer, `<table border="1">`)
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>Image Stream</th>")
		fmt.Fprintln(writer, "    <th>Error</th>")
		fmt.Fprintln(writer, "    <th>Build log</th>")
		fmt.Fprintln(writer, "  </tr>")
		for _, streamName := range streamNames {
			result := failedBuilds[streamName]
			fmt.Fprintf(writer, "  <tr>\n")
			fmt.Fprintf(writer, "    <td>%s</td>\n", streamName)
			fmt.Fprintf(writer, "    <td>%s</td>\n", result.error)
			fmt.Fprintf(writer,
				"    <td><a href=\"showLastBuildLog?%s\">log</a></td>\n",
				streamName)
			fmt.Fprintf(writer, "  </tr>\n")
		}
		fmt.Fprintln(writer, "</table><br>")
	}
	if len(goodBuilds) > 0 {
		streamNames := make([]string, 0, len(goodBuilds))
		for streamName := range goodBuilds {
			streamNames = append(streamNames, streamName)
		}
		sort.Strings(streamNames)
		fmt.Fprintln(writer, "Successful image builds:<br>")
		fmt.Fprintln(writer, `<table border="1">`)
		fmt.Fprintln(writer, "  <tr>")
		fmt.Fprintln(writer, "    <th>Image Stream</th>")
		fmt.Fprintln(writer, "    <th>Name</th>")
		fmt.Fprintln(writer, "    <th>Build log</th>")
		fmt.Fprintln(writer, "  </tr>")
		for _, streamName := range streamNames {
			result := goodBuilds[streamName]
			fmt.Fprintf(writer, "  <tr>\n")
			fmt.Fprintf(writer, "    <td>%s</td>\n", streamName)
			fmt.Fprintf(writer,
				"    <td><a href=\"http://%s/showImage?%s\">%s</a></td>\n",
				b.imageServerAddress, result.imageName, result.imageName)
			fmt.Fprintf(writer,
				"    <td><a href=\"showLastBuildLog?%s\">log</a></td>\n",
				streamName)
			fmt.Fprintf(writer, "  </tr>\n")
		}
		fmt.Fprintln(writer, "</table><br>")
	}
}
