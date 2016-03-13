package httpd

import (
	"fmt"
	"github.com/Symantec/Dominator/imageserver/scanner"
	"github.com/Symantec/Dominator/lib/objectserver/filesystem"
	"io"
	"net"
	"net/http"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

var htmlWriters []HtmlWriter

type state struct {
	imageDataBase *scanner.ImageDataBase
	objectServer  *filesystem.ObjectServer
}

func StartServer(portNum uint, imdb *scanner.ImageDataBase,
	objSrv *filesystem.ObjectServer, daemon bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	myState := state{imageDataBase: imdb, objectServer: objSrv}
	http.HandleFunc("/", statusHandler)
	http.HandleFunc("/listBuildLog", myState.listBuildLogHandler)
	http.HandleFunc("/listComputedInodes", myState.listComputedInodesHandler)
	http.HandleFunc("/listFilter", myState.listFilterHandler)
	http.HandleFunc("/listImage", myState.listImageHandler)
	http.HandleFunc("/listImages", myState.listImagesHandler)
	http.HandleFunc("/listReleaseNotes", myState.listReleaseNotesHandler)
	http.HandleFunc("/listTriggers", myState.listTriggersHandler)
	http.HandleFunc("/showImage", myState.showImageHandler)
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
