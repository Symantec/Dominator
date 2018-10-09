package httpd

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/Symantec/Dominator/imageserver/scanner"
	"github.com/Symantec/Dominator/lib/html"
	"github.com/Symantec/Dominator/lib/objectserver/filesystem"
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
	html.HandleFunc("/", statusHandler)
	html.HandleFunc("/listBuildLog", myState.listBuildLogHandler)
	html.HandleFunc("/listComputedInodes", myState.listComputedInodesHandler)
	html.HandleFunc("/listDirectories", myState.listDirectoriesHandler)
	html.HandleFunc("/listFilter", myState.listFilterHandler)
	html.HandleFunc("/listImage", myState.listImageHandler)
	html.HandleFunc("/listImages", myState.listImagesHandler)
	html.HandleFunc("/listPackages", myState.listPackagesHandler)
	html.HandleFunc("/listReleaseNotes", myState.listReleaseNotesHandler)
	html.HandleFunc("/listTriggers", myState.listTriggersHandler)
	html.HandleFunc("/showImage", myState.showImageHandler)
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
