package scanner

import (
	"fmt"
	"io"
)

func (imdb *ImageDataBase) writeHtml(writer io.Writer) {
	fmt.Fprintf(writer, "Number of images: <a href=\"listImages\">%d</a><br>\n",
		imdb.CountImages())
}
