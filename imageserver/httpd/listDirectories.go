package httpd

import (
	"bufio"
	"fmt"
	"io"
	"net/http"

	"github.com/Symantec/Dominator/lib/image"
)

func (s state) listDirectoriesHandler(w http.ResponseWriter,
	req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	directories := s.imageDataBase.ListDirectories()
	image.SortDirectories(directories)
	if req.URL.RawQuery == "output=text" {
		for _, directory := range directories {
			fmt.Fprintln(writer, directory.Name)
		}
		return
	}
	fmt.Fprintln(writer, "<title>imageserver directories</title>")
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	fmt.Fprintln(writer, `<table border="1" style="width:100%">`)
	fmt.Fprintln(writer, "  <tr>")
	fmt.Fprintln(writer, "    <th>Name</th>")
	fmt.Fprintln(writer, "    <th>Owner Group</th>")
	fmt.Fprintln(writer, "  </tr>")
	for _, directory := range directories {
		showDirectory(writer, directory)
	}
	fmt.Fprintln(writer, "</table>")
	fmt.Fprintln(writer, "</body>")
}

func showDirectory(writer io.Writer, directory image.Directory) {
	fmt.Fprintf(writer, "  <tr>\n")
	fmt.Fprintf(writer, "    <td>%s</td>\n", directory.Name)
	fmt.Fprintf(writer, "    <td>%s</td>\n", directory.Metadata.OwnerGroup)
	fmt.Fprintf(writer, "  </tr>\n")
}
