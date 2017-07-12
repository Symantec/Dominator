package httpd

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectserver/filesystem"
	"io"
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
