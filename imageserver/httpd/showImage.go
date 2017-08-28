package httpd

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/image"
	"io"
	"net/http"
	"time"
)

var timeFormat string = "02 Jan 2006 15:04:05.99 MST"

func (s state) showImageHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	imageName := req.URL.RawQuery
	fmt.Fprintf(writer, "<title>image %s</title>\n", imageName)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	image := s.imageDataBase.GetImage(imageName)
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
	if numInodes := image.FileSystem.NumComputedRegularInodes(); numInodes > 0 {
		fmt.Fprintf(writer,
			"Number of computed inodes: <a href=\"listComputedInodes?%s\">%d</a><br>\n",
			imageName, numInodes)
	}
	if image.Filter == nil {
		fmt.Fprintln(writer, "Image has no filter: sparse image<br>")
	} else if len(image.Filter.FilterLines) < 1 {
		fmt.Fprintln(writer,
			"Filter has 0 lines (empty filter: full coverage)<br>")
	} else {
		fmt.Fprintf(writer,
			"Filter has <a href=\"listFilter?%s\">%d</a> lines<br>\n",
			imageName, len(image.Filter.FilterLines))
	}
	if image.Triggers == nil || len(image.Triggers.Triggers) < 1 {
		fmt.Fprintln(writer, "Image has no triggers<br>")
	} else {
		fmt.Fprintf(writer,
			"Number of triggers: <a href=\"listTriggers?%s\">%d</a><br>\n",
			imageName, len(image.Triggers.Triggers))
	}
	if !image.ExpiresAt.IsZero() {
		fmt.Fprintf(writer, "Expires at: %s (in %s)<br>\n",
			image.ExpiresAt.In(time.Local).Format(timeFormat),
			format.Duration(time.Until(image.ExpiresAt)))
	}
	showAnnotation(writer, image.ReleaseNotes, imageName, "Release notes",
		"listReleaseNotes")
	showAnnotation(writer, image.BuildLog, imageName, "Build log",
		"listBuildLog")
	if image.CreatedBy != "" {
		fmt.Fprintf(writer, "Created by: %s\n<br>", image.CreatedBy)
	}
	if !image.CreatedOn.IsZero() {
		fmt.Fprintf(writer, "Created on: %s (%s old)\n<br>",
			image.CreatedOn.In(time.Local).Format(timeFormat),
			format.Duration(time.Since(image.CreatedOn)))
	}
	if len(image.Packages) > 0 {
		fmt.Fprintf(writer,
			"Packages: <a href=\"listPackages?%s\">%d</a><br>\n",
			imageName, len(image.Packages))
	}
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
