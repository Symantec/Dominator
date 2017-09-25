package httpd

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/url"
)

func (s state) listPackagesHandler(w http.ResponseWriter, req *http.Request) {
	parsedQuery := url.ParseQuery(req.URL)
	if len(parsedQuery.Flags) != 1 {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var imageName string
	for name := range parsedQuery.Flags {
		imageName = name
	}
	image := s.imageDataBase.GetImage(imageName)
	if image == nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	switch parsedQuery.OutputType() {
	case url.OutputTypeText:
		for _, pkg := range image.Packages {
			fmt.Fprintln(writer, pkg.Name, pkg.Version, pkg.Size)
		}
		return
	case url.OutputTypeJson:
		err := json.WriteWithIndent(writer, "    ", image.Packages)
		if err != nil {
			fmt.Fprintln(writer, err)
		}
		return
	case url.OutputTypeHtml:
		break
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Fprintf(writer, "<title>image %s packages</title>\n", imageName)
	fmt.Fprintln(writer, `<style>
                          table, th, td {
                          border-collapse: collapse;
                          }
                          </style>`)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	fmt.Fprintf(writer, "Packages in image: %s", imageName)
	fmt.Fprintf(writer, " <a href=\"listPackages?%s&output=text\">text</a>",
		imageName)
	fmt.Fprintf(writer, " <a href=\"listPackages?%s&output=json\">json</a>",
		imageName)
	fmt.Fprintln(writer, "</h3>")
	fmt.Fprintln(writer, `<table border="1" style="width:100%">`)
	fmt.Fprintln(writer, "  <tr>")
	fmt.Fprintln(writer, "    <th>Name</th>")
	fmt.Fprintln(writer, "    <th>Version</th>")
	fmt.Fprintln(writer, "    <th>Size</th>")
	fmt.Fprintln(writer, "  </tr>")
	for _, pkg := range image.Packages {
		fmt.Fprintf(writer, "  <tr>\n")
		fmt.Fprintf(writer, "    <td>%s</td>\n", pkg.Name)
		fmt.Fprintf(writer, "    <td>%s</td>\n", pkg.Version)
		fmt.Fprintf(writer, "    <td>%s</td>\n", format.FormatBytes(pkg.Size))
		fmt.Fprintf(writer, "  </tr>\n")
	}
	fmt.Fprintln(writer, "</table>")
	fmt.Fprintln(writer, "</body>")
}
