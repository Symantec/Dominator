package httpd

import (
	"bufio"
	"fmt"
	"net/http"
)

func (s *state) listGeneratorsHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	for _, pathname := range s.manager.GetRegisteredPaths() {
		fmt.Fprintln(writer, pathname)
	}
}
