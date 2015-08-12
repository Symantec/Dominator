package rpcd

import (
	"github.com/Symantec/Dominator/proto/objectserver"
	"runtime"
)

func (t *rpcType) AddObjects(request objectserver.AddObjectsRequest,
	reply *objectserver.AddObjectsResponse) error {
	var response objectserver.AddObjectsResponse
	for _, objectToAdd := range request.ObjectsToAdd {
		hash, err := objectServer.AddObject(objectToAdd.ObjectData,
			objectToAdd.ExpectedHash)
		if err != nil {
			return err
		}
		response.Hashes = append(response.Hashes, hash)
	}
	*reply = response
	runtime.GC() // An opportune time to take out the garbage.
	return nil
}
