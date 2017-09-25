package lib

import (
	"encoding/gob"
	"io"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
)

func addObjects(conn *srpc.Conn, adder ObjectAdder, logger log.Logger) error {
	defer conn.Flush()
	logger.Printf("AddObjects(%s) starting\n", conn.RemoteAddr())
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
	numAdded := 0
	numObj := 0
	for ; ; numObj++ {
		var request objectserver.AddObjectRequest
		var response objectserver.AddObjectResponse
		if err := decoder.Decode(&request); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return errors.New("error decoding: " + err.Error())
		}
		if request.Length < 1 {
			break
		}
		var err error
		response.Hash, response.Added, err =
			adder.AddObject(conn, request.Length, request.ExpectedHash)
		response.ErrorString = errors.ErrorToString(err)
		if err := encoder.Encode(response); err != nil {
			return errors.New("error encoding: " + err.Error())
		}
		if response.ErrorString != "" {
			logger.Printf(
				"AddObjects(): failed, %d of %d so far are new objects: %s",
				numAdded, numObj+1, response.ErrorString)
			return nil
		}
		if response.Added {
			numAdded++
		}
	}
	logger.Printf("AddObjects(): %d of %d are new objects", numAdded, numObj)
	return nil
}
