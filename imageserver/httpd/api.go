package httpd

import (
	"fmt"
	"github.com/Symantec/Dominator/imageserver/scanner"
	"github.com/Symantec/Dominator/objectserver/filesystem"
	"io"
	"net"
	"net/http"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

var htmlWriters []HtmlWriter

var imageDataBase *scanner.ImageDataBase
var objectServer *filesystem.ObjectServer

func StartServer(portNum uint, imdb *scanner.ImageDataBase,
	objSrv *filesystem.ObjectServer, daemon bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	imageDataBase = imdb
	objectServer = objSrv
	http.HandleFunc("/", statusHandler)
	http.HandleFunc("/listBuildLog", listBuildLogHandler)
	http.HandleFunc("/listFilter", listFilterHandler)
	http.HandleFunc("/listImage", listImageHandler)
	http.HandleFunc("/listImages", listImagesHandler)
	http.HandleFunc("/listReleaseNotes", listReleaseNotesHandler)
	http.HandleFunc("/listTriggers", listTriggersHandler)
	http.HandleFunc("/showImage", showImageHandler)
	if daemon {
		go http.Serve(listener, nil)
	} else {
		http.Serve(listener, nil)
	}
	return nil
}

func AddHtmlWriter(htmlWriter HtmlWriter) {
	htmlWriters = append(htmlWriters, htmlWriter)
}
