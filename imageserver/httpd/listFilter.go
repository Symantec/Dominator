package httpd

import (
	"bufio"
	"fmt"
	"net/http"
)

func listFilterHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	imageName := req.URL.RawQuery
	fmt.Fprintf(writer, "<title>filter %s</title>\n", imageName)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	writeLinks(writer)
	image := imageDataBase.GetImage(imageName)
	if image == nil {
		fmt.Fprintf(writer, "Image: %s UNKNOWN!\n", imageName)
	} else {
		fmt.Fprintf(writer, "Filter lines for image: %s\n", imageName)
		fmt.Fprintln(writer, "<pre>")
		for _, line := range image.Filter.FilterLines {
			fmt.Fprintln(writer, line)
		}
		fmt.Fprintln(writer, "</pre>")
	}
	fmt.Fprintln(writer, "</body>")
}
