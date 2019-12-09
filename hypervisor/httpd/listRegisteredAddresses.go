package httpd

import (
	"bufio"
	"net/http"

	"github.com/Cloud-Foundations/Dominator/lib/json"
)

func (s state) listRegisteredAddressesHandler(w http.ResponseWriter,
	req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	addresses := s.manager.ListRegisteredAddresses()
	json.WriteWithIndent(writer, "    ", addresses)
}
