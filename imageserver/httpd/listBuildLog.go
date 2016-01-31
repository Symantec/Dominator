package httpd

import (
	"bufio"
	"fmt"
	"net/http"
)

func listBuildLogHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	imageName := req.URL.RawQuery
	fmt.Fprintf(writer, "<title>image %s</title>\n", imageName)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	image := imageDataBase.GetImage(imageName)
	if image == nil {
		fmt.Fprintf(writer, "Image: %s UNKNOWN!\n", imageName)
		return
	}
	if image.BuildLog == nil {
		fmt.Fprintf(writer, "No build log for image: %s\n", imageName)
		return
	}
	if image.BuildLog.Object == nil {
		fmt.Fprintf(writer, "No build log data for image: %s\n", imageName)
		return
	}
	fmt.Fprintf(writer, "Build log for image: %s<br>\n", imageName)
	fmt.Fprintln(writer, "</h3>")
	listObject(writer, objectServer, image.BuildLog.Object)
	fmt.Fprintln(writer, "</body>")
}
