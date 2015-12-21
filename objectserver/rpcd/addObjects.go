package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
	"runtime"
)

func (t *srpcType) AddObjects(conn *srpc.Conn) {
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
			response.Error = err
		} else if request.Length < 1 {
			break
		} else {
			response.Hash, response.Added, response.Error =
				t.objectServer.AddObject(
					conn, request.Length, request.ExpectedHash)
			if response.Added {
				numAdded++
			}
		}
		encoder.Encode(response)
		if response.Error != nil {
			return
		}
	}
	t.logger.Printf("AddObjects(): %d of %d are new objects", numAdded, numObj)
}
