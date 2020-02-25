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
	if len(columns) > 0 {
		if doHighlighting {
			fmt.Fprintf(writer, "  <tr style=\"background-color:%s\">\n",
				headingBackground)
		} else {
			fmt.Fprintln(writer, "  <tr>")
		}
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

func (tw *TableWriter) closeRow() error {
	_, err := fmt.Fprintln(tw.writer, "  </tr>")
	return err
}

func (tw *TableWriter) openRow(foreground, background string) error {
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
	var err error
	if background == defaultBackground {
		if foreground == defaultForeground {
			_, err = fmt.Fprintln(tw.writer, "  <tr>")
		} else {
			_, err = fmt.Fprintf(tw.writer, "  <tr style=\"color:%s\">\n",
				foreground)
		}
	} else {
		if foreground == defaultForeground {
			_, err = fmt.Fprintf(tw.writer,
				"  <tr style=\"background-color:%s\">\n", background)
		} else {
			_, err = fmt.Fprintf(tw.writer,
				"  <tr style=\"background-color:%s;color:%s\">\n",
				background, foreground)
		}
	}
	tw.lastBackground = background
	return err
}

func (tw *TableWriter) writeData(foreground, data string) error {
	if foreground == "" {
		foreground = defaultForeground
	}
	var err error
	if foreground == defaultForeground {
		_, err = fmt.Fprintf(tw.writer, "    <td>%s</td>\n", data)
	} else {
		_, err = fmt.Fprintf(tw.writer,
			"    <td><font color=\"%s\">%s</font></td>\n",
			foreground, data)
	}
	return err
}

func (tw *TableWriter) writeRow(foreground, background string,
	columns []string) error {
	if err := tw.OpenRow(foreground, background); err != nil {
		return err
	}
	for _, column := range columns {
		fmt.Fprintf(tw.writer, "    <td>%s</td>\n", column)
	}
	return tw.CloseRow()
}
