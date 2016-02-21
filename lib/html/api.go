package html

import (
	"io"
	"net/http"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

func WriteHeaderWithRequest(writer io.Writer, req *http.Request) {
	writeHeader(writer, req)
}

func WriteHeader(writer io.Writer) {
	writeHeader(writer, nil)
}

func WriteFooter(writer io.Writer) {
	writeFooter(writer)
}

func RegisterHtmlWriterForPattern(pattern, title string,
	htmlWriter HtmlWriter) {
	registerHtmlWriterForPattern(pattern, title, htmlWriter)
}
