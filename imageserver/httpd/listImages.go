package httpd

import (
	"bufio"
	"fmt"
	"net/http"
)

func listImagesHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	fmt.Fprintln(writer, "<title>imageserver images</title>")
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	writeLinks(writer)
	images := imageDataBase.ListImages()
	for _, image := range images {
		fmt.Fprintf(writer, "%s<br>\n", image)
	}
	fmt.Fprintln(writer, "</body>")
}
