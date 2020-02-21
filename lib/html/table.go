package html

import (
	"fmt"
	"io"
)

const (
	defaultBackground   = "white"
	defaultForeground   = "black"
	headingBackground   = "#f0f0f0"
	highlightBackground = "#fafafa"
)

func newTableWriter(writer io.Writer, doHighlighting bool,
	columns []string) (*TableWriter, error) {
	if doHighlighting {
		fmt.Fprintf(writer, "  <tr style=\"background-color:%s\">\n",
			headingBackground)
	} else {
		fmt.Fprintln(writer, "  <tr>")
	}
	for _, column := range columns {
		fmt.Fprintf(writer, "    <th>%s</th>\n", column)
	}
	fmt.Fprintln(writer, "  </tr>")
	return &TableWriter{
		doHighlighting: doHighlighting,
		writer:         writer,
	}, nil
}

func (tw *TableWriter) writeRow(foreground, background string,
	columns []string) error {
	if foreground == "" {
		foreground = defaultForeground
	}
	if background == "" {
		background = defaultBackground
	}
	if background == defaultBackground &&
		tw.lastBackground == defaultBackground {
		background = highlightBackground
	}
	if background == defaultBackground {
		if foreground == defaultForeground {
			fmt.Fprintln(tw.writer, "  <tr>")
		} else {
			fmt.Fprintf(tw.writer, "  <tr style=\"color:%s\">\n", foreground)
		}
	} else {
		if foreground == defaultForeground {
			fmt.Fprintf(tw.writer, "  <tr style=\"background-color:%s\">\n",
				background)
		} else {
			fmt.Fprintf(tw.writer,
				"  <tr style=\"background-color:%s;color:%s\">\n",
				background, foreground)
		}
	}
	for _, column := range columns {
		fmt.Fprintf(tw.writer, "    <td>%s</td>\n", column)
	}
	fmt.Fprintln(tw.writer, "  </tr>")
	tw.lastBackground = background
	return nil
}
