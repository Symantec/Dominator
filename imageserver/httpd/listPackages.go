package httpd

import (
	"bufio"
	"fmt"
	"net/http"

	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/html"
	"github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/url"
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
	tw, _ := html.NewTableWriter(writer, true, "Name", "Version", "Size")
	for _, pkg := range image.Packages {
		tw.WriteRow("", "",
			pkg.Name,
			pkg.Version,
			format.FormatBytes(pkg.Size),
		)
	}
	fmt.Fprintln(writer, "</table>")
	fmt.Fprintln(writer, "</body>")
}
