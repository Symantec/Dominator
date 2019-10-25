package cachingreader

import (
	"fmt"
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/format"
)

func (objSrv *ObjectServer) writeHtml(writer io.Writer) {
	objSrv.rwLock.RLock()
	defer objSrv.rwLock.RUnlock()
	fmt.Fprintf(writer,
		"Objectcache max: %s, total: %s (%d), cached: %s, in use: %s, downloading: %s<br>\n",
		format.FormatBytes(objSrv.maxCachedBytes),
		format.FormatBytes(objSrv.cachedBytes+objSrv.downloadingBytes),
		len(objSrv.objects),
		format.FormatBytes(objSrv.cachedBytes),
		format.FormatBytes(objSrv.cachedBytes-objSrv.lruBytes),
		format.FormatBytes(objSrv.downloadingBytes))
}
