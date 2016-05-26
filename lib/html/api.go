package html

import (
	"io"
	"net/http"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

func WriteHeaderWithRequest(writer io.Writer, req *http.Request) {
	header := Header{Request: req}
	header.WriteHtml(writer)
}

func WriteHeader(writer io.Writer) {
	var header Header
	header.WriteHtml(writer)
}

func WriteFooter(writer io.Writer) {
	writeFooter(writer)
}

func RegisterHtmlWriterForPattern(pattern, title string,
	htmlWriter HtmlWriter) {
	registerHtmlWriterForPattern(pattern, title, htmlWriter)
}

type Header struct {
	NoGC    bool
	Request *http.Request
}

func (h *Header) WriteHtml(writer io.Writer) {
	h.writeHtml(writer)
}
