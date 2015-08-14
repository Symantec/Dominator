package rpcd

import (
	"github.com/Symantec/Dominator/proto/objectserver"
	"runtime"
)

func (t *rpcType) AddObjects(request objectserver.AddObjectsRequest,
	reply *objectserver.AddObjectsResponse) error {
	var response objectserver.AddObjectsResponse
	var err error
	response.Hashes, err = objectServer.AddObjects(request.ObjectDatas,
		request.ExpectedHashes)
	if err != nil {
		return err
	}
	*reply = response
	runtime.GC() // An opportune time to take out the garbage.
	return nil
}
