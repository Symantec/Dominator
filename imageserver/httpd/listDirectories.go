package httpd

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/Cloud-Foundations/Dominator/lib/html"
	"github.com/Cloud-Foundations/Dominator/lib/image"
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
	tw, _ := html.NewTableWriter(writer, true, "Name", "Owner Group")
	for _, directory := range directories {
		tw.WriteRow("", "", directory.Name, directory.Metadata.OwnerGroup)
	}
	fmt.Fprintln(writer, "</table>")
	fmt.Fprintln(writer, "</body>")
}
