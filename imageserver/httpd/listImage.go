package httpd

import (
	"bufio"
	"fmt"
	"net/http"
)

func listImageHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	imageName := req.URL.RawQuery
	fmt.Fprintf(writer, "<title>image %s</title>\n", imageName)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	image := imageDataBase.GetImage(imageName)
	if image == nil {
		fmt.Fprintf(writer, "Image: %s UNKNOWN!\n", imageName)
	} else {
		fmt.Fprintf(writer, "File-system data for image: %s\n", imageName)
		fmt.Fprintln(writer, "</h3>")
		fmt.Fprintln(writer, "<pre>")
		image.FileSystem.List(writer)
		fmt.Fprintln(writer, "</pre>")
	}
	fmt.Fprintln(writer, "</body>")
}
