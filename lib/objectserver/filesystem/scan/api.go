package scan

import (
	"github.com/Symantec/Dominator/lib/hash"
)

// ScanTree will scan a directory tree for objects and will call registerFunc
// for each object. Multiple calls to registerFunc may be called concurrently.
func ScanTree(baseDir string, registerFunc func(hash.Hash, uint64)) error {
	return scanTree(baseDir, registerFunc)
}
