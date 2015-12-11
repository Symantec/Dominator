package httpd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func listTriggersHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	imageName := req.URL.RawQuery
	fmt.Fprintf(writer, "<title>triggers %s</title>\n", imageName)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	image := imageDataBase.GetImage(imageName)
	if image == nil {
		fmt.Fprintf(writer, "Image: %s UNKNOWN!\n", imageName)
	} else {
		fmt.Fprintf(writer, "Triggers for image: %s\n", imageName)
		fmt.Fprintln(writer, "<pre>")
		b, _ := json.Marshal(image.Triggers.Triggers)
		var out bytes.Buffer
		json.Indent(&out, b, "", "    ")
		out.WriteTo(writer)
		fmt.Fprintln(writer, "</pre>")
	}
	fmt.Fprintln(writer, "</body>")
}
