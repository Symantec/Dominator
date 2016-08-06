package httpd

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/json"
	"net/http"
)

func (s state) listTriggersHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	imageName := req.URL.RawQuery
	fmt.Fprintf(writer, "<title>triggers %s</title>\n", imageName)
	fmt.Fprintln(writer, "<body>")
	fmt.Fprintln(writer, "<h3>")
	image := s.imageDataBase.GetImage(imageName)
	if image == nil {
		fmt.Fprintf(writer, "Image: %s UNKNOWN!\n", imageName)
	} else if image.Triggers == nil {
		fmt.Fprintln(writer, "NO TRIGGERS")
	} else {
		fmt.Fprintf(writer, "Triggers for image: %s\n", imageName)
		fmt.Fprintln(writer, "<pre>")
		json.WriteWithIndent(writer, "    ", image.Triggers.Triggers)
		fmt.Fprintln(writer, "</pre>")
	}
	fmt.Fprintln(writer, "</body>")
}
