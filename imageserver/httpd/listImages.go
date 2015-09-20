package httpd

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/image"
	"io"
	"net/http"
	"sort"
)

func listImagesHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>imageserver images</title>")
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	writeLinks(writer)
	imageNames := imageDataBase.ListImages()
	sort.Strings(imageNames)
	fmt.Fprintln(writer, `<table border="1" style="width:100%">`)
	fmt.Fprintln(writer, "  <tr>")
	fmt.Fprintln(writer, "    <th>Name</b></th>")
	fmt.Fprintln(writer, "    <th>Data Size</b></th>")
	fmt.Fprintln(writer, "    <th>Data Inodes</b></th>")
	fmt.Fprintln(writer, "    <th>Filter Lines</b></th>")
	fmt.Fprintln(writer, "    <th>Triggers</b></th>")
	fmt.Fprintln(writer, "  </tr>")
	for _, name := range imageNames {
		showImage(writer, name, imageDataBase.GetImage(name))
	}
	fmt.Fprintln(writer, "</table>")
	fmt.Fprintln(writer, "</body>")
}

func showImage(writer io.Writer, name string, image *image.Image) {
	fmt.Fprintf(writer, "  <tr>\n")
	fmt.Fprintf(writer, "    <td>%s</td>\n", name)
	fmt.Fprintf(writer, "    <td><a href=\"listImage?%s\">%s</a></td>\n",
		name, format.FormatBytes(image.FileSystem.TotalDataBytes))
	fmt.Fprintf(writer, "    <td><a href=\"listImage?%s\">%d</a></td>\n",
		name, len(image.FileSystem.RegularInodeTable))
	fmt.Fprintf(writer, "    <td><a href=\"listFilter?%s\">%d</a></td>\n",
		name, len(image.Filter.FilterLines))
	fmt.Fprintf(writer, "    <td><a href=\"listTriggers?%s\">%d</a></td>\n",
		name, len(image.Triggers.Triggers))
	fmt.Fprintf(writer, "  </tr>\n")
}
