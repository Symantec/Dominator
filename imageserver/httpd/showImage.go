package httpd

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/image"
	"io"
	"net/http"
)

func showImageHandler(w http.ResponseWriter, req *http.Request) {
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
	fmt.Fprintf(writer, "Information for image: %s<br>\n", imageName)
	fmt.Fprintln(writer, "</h3>")
	fmt.Fprintf(writer, "Data size: <a href=\"listImage?%s\">%s</a><br>\n",
		imageName, format.FormatBytes(image.FileSystem.TotalDataBytes))
	fmt.Fprintf(writer, "Number of data inodes: %d<br>\n",
		image.FileSystem.NumRegularInodes)
	if image.Filter == nil {
		fmt.Fprintln(writer, "Image has no filter: sparse image<br>")
	} else {
		fmt.Fprintf(writer,
			"Filter has <a href=\"listFilter?%s\">%d</a> lines<br>\n",
			imageName, len(image.Filter.FilterLines))
	}
	fmt.Fprintf(writer,
		"Number of triggers: <a href=\"listTriggers?%s\">%d</a><br>\n",
		imageName, len(image.Triggers.Triggers))
	showAnnotation(writer, image.ReleaseNotes, imageName, "Release notes",
		"listReleaseNotes")
	showAnnotation(writer, image.BuildLog, imageName, "Build log",
		"listBuildLog")
	fmt.Fprintln(writer, "</body>")
}

func showAnnotation(writer io.Writer, annotation *image.Annotation,
	imageName string, linkName string, baseURL string) {
	if annotation == nil {
		return
	}
	var url string
	if annotation.URL != "" {
		url = annotation.URL
	} else {
		url = baseURL + "?" + imageName
	}
	fmt.Fprintf(writer, "<a href=\"%s\">%s</a><br>\n", url, linkName)
}
