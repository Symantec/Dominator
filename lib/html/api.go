package html

import (
	"io"
	"net/http"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

func BenchmarkedHandler(handler func(io.Writer,
	*http.Request)) func(http.ResponseWriter, *http.Request) {
	return benchmarkedHandler(handler)
}

func HandleFunc(pattern string,
	handler func(w http.ResponseWriter, req *http.Request)) {
	handleFunc(http.DefaultServeMux, pattern, handler)
}

func RegisterHtmlWriterForPattern(pattern, title string,
	htmlWriter HtmlWriter) {
	registerHtmlWriterForPattern(pattern, title, htmlWriter)
}

func ServeMuxHandleFunc(serveMux *http.ServeMux, pattern string,
	handler func(w http.ResponseWriter, req *http.Request)) {
	handleFunc(serveMux, pattern, handler)
}

func SetSecurityHeaders(w http.ResponseWriter) {
	setSecurityHeaders(w)
}

func WriteFooter(writer io.Writer) {
	writeFooter(writer)
}

func WriteHeader(writer io.Writer) {
	writeHeader(writer, nil, false)
}

func WriteHeaderNoGC(writer io.Writer) {
	writeHeader(writer, nil, true)
}

func WriteHeaderWithRequest(writer io.Writer, req *http.Request) {
	writeHeader(writer, req, false)
}

func WriteHeaderWithRequestNoGC(writer io.Writer, req *http.Request) {
	writeHeader(writer, req, true)
}
