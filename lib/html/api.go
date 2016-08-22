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

func RegisterHtmlWriterForPattern(pattern, title string,
	htmlWriter HtmlWriter) {
	registerHtmlWriterForPattern(pattern, title, htmlWriter)
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
