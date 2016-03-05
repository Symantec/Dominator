package lib

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"io"
	"log"
)

type ObjectAdder interface {
	AddObject(reader io.Reader, length uint64, expectedHash *hash.Hash) (
		hash.Hash, bool, error)
}

func AddObjects(conn *srpc.Conn, adder ObjectAdder, logger *log.Logger) error {
	return addObjects(conn, adder, logger)
}
