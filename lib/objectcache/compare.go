package objectcache

import (
	"bytes"
	"fmt"
	"io"
)

func compareObjects(left, right ObjectCache, logWriter io.Writer) bool {
	if len(left) != len(right) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, "left vs. right: %d vs. %d objects\n",
				len(left), len(right))
		}
		return false
	}
	for index, leftHash := range left {
		if bytes.Compare(leftHash[:], right[index][:]) != 0 {
			if logWriter != nil {
				fmt.Fprintf(logWriter, "hash: left vs. right: %x vs. %x\n",
					leftHash, right[index])
			}
			return false
		}
	}
	return true
}
