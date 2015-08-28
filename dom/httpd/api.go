package httpd

import (
	"fmt"
	"net"
	"net/http"
)

func StartServer(portNum uint, daemon bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	http.HandleFunc("/", statusHandler)
	if daemon {
		go http.Serve(listener, nil)
	} else {
		http.Serve(listener, nil)
	}
	return nil
}
