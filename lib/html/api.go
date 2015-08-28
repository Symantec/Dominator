package html

import (
	"io"
)

func WriteHeader(writer io.Writer) {
	writeHeader(writer)
}

func WriteFooter(writer io.Writer) {
	writeFooter(writer)
}
