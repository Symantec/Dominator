package scanner

import (
	"fmt"
	"io"
)

func (imdb *ImageDataBase) writeHtml(writer io.Writer) {
	fmt.Fprintf(writer,
		"Number of  <a href=\"listImages?output=text\">images</a>: "+
			"<a href=\"listImages\">%d</a><br>\n",
		imdb.CountImages())
}
