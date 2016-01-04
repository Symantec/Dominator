package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
	"runtime"
)

func (t *srpcType) AddObjects(conn *srpc.Conn) error {
	defer runtime.GC() // An opportune time to take out the garbage.
	defer conn.Flush()
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
			return err
		}
		if request.Length < 1 {
			break
		}
		response.Hash, response.Added, response.Error =
			t.objectServer.AddObject(conn, request.Length, request.ExpectedHash)
		if response.Added {
			numAdded++
		}
		if err := encoder.Encode(response); err != nil {
			return err
		}
		if response.Error != nil {
			t.logger.Printf("AddObjects(): failed, %d of %d are new objects %s",
				numAdded, numObj, response.Error.Error())
			return nil
		}
	}
	t.logger.Printf("AddObjects(): %d of %d are new objects", numAdded, numObj)
	return nil
}
