package httpd

import (
	"fmt"
	"github.com/Symantec/Dominator/imageserver/scanner"
	"io"
	"net"
	"net/http"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

var imageDataBase *scanner.ImageDataBase

func StartServer(portNum uint, imdb *scanner.ImageDataBase, daemon bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	imageDataBase = imdb
	http.HandleFunc("/", statusHandler)
	http.HandleFunc("/listFilter", listFilterHandler)
	http.HandleFunc("/listImage", listImageHandler)
	http.HandleFunc("/listImages.html", listImagesHandler)
	if daemon {
		go http.Serve(listener, nil)
	} else {
		http.Serve(listener, nil)
	}
	return nil
}
