package html

import (
	"io"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

func WriteHeader(writer io.Writer) {
	writeHeader(writer)
}

func WriteFooter(writer io.Writer) {
	writeFooter(writer)
}

func RegisterHtmlWriterForPattern(pattern, title string,
	htmlWriter HtmlWriter) {
	registerHtmlWriterForPattern(pattern, title, htmlWriter)
}
