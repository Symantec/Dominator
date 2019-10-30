package lib

import (
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

type ObjectAdder interface {
	AddObject(reader io.Reader, length uint64, expectedHash *hash.Hash) (
		hash.Hash, bool, error)
}

func AddObjects(conn *srpc.Conn, decoder srpc.Decoder, encoder srpc.Encoder,
	adder ObjectAdder, logger log.Logger) error {
	return addObjects(conn, decoder, encoder, adder, logger)
}

func AddObjectsWithMaster(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder, objSrv objectserver.StashingObjectServer,
	masterAddress string, logger log.DebugLogger) error {
	return addObjectsWithMaster(conn, decoder, encoder, objSrv, masterAddress,
		logger)
}
