package httpd

import (
	"fmt"
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver/filesystem"
)

func listObject(writer io.Writer, objSrv *filesystem.ObjectServer,
	hashP *hash.Hash) {
	_, reader, err := objSrv.GetObject(*hashP)
	if err != nil {
		fmt.Fprintln(writer, err)
		return
	}
	defer reader.Close()
	fmt.Fprintln(writer, "<pre>")
	io.Copy(writer, reader)
	fmt.Fprintln(writer, "</pre>")
}
