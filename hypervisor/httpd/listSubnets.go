package httpd

import (
	"bufio"
	"net/http"

	"github.com/Cloud-Foundations/Dominator/lib/json"
)

func (s state) listSubnetsHandler(w http.ResponseWriter, req *http.Request) {
	writer := bufio.NewWriter(w)
	defer writer.Flush()
	subnets := s.manager.ListSubnets(true)
	json.WriteWithIndent(writer, "    ", subnets)
}
