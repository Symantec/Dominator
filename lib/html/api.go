package html

import (
	"io"
	"net/http"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

func WriteHeaderWithRequest(writer io.Writer, req *http.Request) {
	writeHeader(writer, req, false)
}

func WriteHeader(writer io.Writer) {
	writeHeader(writer, nil, false)
}

func WriteHeaderNoGC(writer io.Writer) {
	writeHeader(writer, nil, true)
}

func WriteFooter(writer io.Writer) {
	writeFooter(writer)
}

func RegisterHtmlWriterForPattern(pattern, title string,
	htmlWriter HtmlWriter) {
	registerHtmlWriterForPattern(pattern, title, htmlWriter)
}
