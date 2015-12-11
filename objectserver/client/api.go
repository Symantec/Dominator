package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
)

func CallAddObjects(client *srpc.Client, request objectserver.AddObjectsRequest,
	reply *objectserver.AddObjectsResponse) error {
	return callAddObjects(client, request, reply)
}
