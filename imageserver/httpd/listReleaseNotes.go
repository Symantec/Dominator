package httpd

import (
	"bufio"
	"fmt"
	"net/http"
)

func listReleaseNotesHandler(w http.ResponseWriter, req *http.Request) {
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
	if image.ReleaseNotes == nil {
		fmt.Fprintf(writer, "No release notes for image: %s\n", imageName)
		return
	}
	if image.ReleaseNotes.Object == nil {
		fmt.Fprintf(writer, "No release notes data for image: %s\n", imageName)
		return
	}
	fmt.Fprintf(writer, "Release notes for image: %s<br>\n", imageName)
	fmt.Fprintln(writer, "</h3>")
	listObject(writer, objectServer, image.ReleaseNotes.Object)
	fmt.Fprintln(writer, "</body>")
}
